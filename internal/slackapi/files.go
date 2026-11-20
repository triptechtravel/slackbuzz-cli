// Hand-written file-upload helper.
//
// Slack hard-deprecated /files.upload (it returns method_deprecated). Its
// replacement is a 3-step external flow; steps 1 and 3 are ordinary API
// methods with generated wrappers (added to the spec in cmd/gen-api/patches.go,
// since Slack's published spec predates them):
//
//  1. FilesGetUploadURLExternal — filename + length → presigned URL + file_id.
//  2. POST the bytes to the presigned URL (multipart, "filename" field). Not a
//     Slack API method and unauthenticated: the only hand-written step. It's
//     streamed, and bounded by the caller's context rather than the client's
//     30s per-request timeout, which a large file on a slow link outlasts.
//  3. FilesCompleteUploadExternal — shares every file_id in ONE call, so all
//     the files arrive as one message with the initial comment.
package slackapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// UploadFileSpec is one file in an UploadFiles call.
type UploadFileSpec struct {
	FilePath string // path on disk; required
	Title    string // defaults to the file's name
}

// UploadFilesParams shares several files as one message.
type UploadFilesParams struct {
	Files          []UploadFileSpec
	ChannelID      string
	InitialComment string
	ThreadTS       string
}

// UploadFiles uploads every file (steps 1-2), then shares them together in a
// single completeUploadExternal call: one message carrying all the files and
// the initial comment. It fails unless Slack reports every file shared.
func UploadFiles(ctx context.Context, c *Client, params *UploadFilesParams) (*FilesCompleteUploadExternalResponse, error) {
	if params == nil || len(params.Files) == 0 {
		return nil, fmt.Errorf("UploadFiles: at least one file is required")
	}
	type fileSpec struct {
		ID    string `json:"id"`
		Title string `json:"title,omitempty"`
	}
	specs := make([]fileSpec, 0, len(params.Files))
	for _, f := range params.Files {
		if f.FilePath == "" {
			return nil, fmt.Errorf("UploadFiles: FilePath is required")
		}
		info, err := os.Stat(f.FilePath)
		if err != nil {
			return nil, fmt.Errorf("UploadFiles: stat %s: %w", f.FilePath, err)
		}
		filename := filepath.Base(f.FilePath)

		step1, err := FilesGetUploadURLExternal(ctx, c, &FilesGetUploadURLExternalParams{
			Filename: filename, Length: int(info.Size()),
		})
		if err != nil {
			return nil, fmt.Errorf("files.getUploadURLExternal: %w", err)
		}
		if step1.UploadURL == "" || step1.FileID == "" {
			return nil, fmt.Errorf("files.getUploadURLExternal: empty upload_url or file_id in response")
		}
		if err := putFileToPresignedURL(ctx, c, step1.UploadURL, f.FilePath, filename); err != nil {
			return nil, err
		}

		title := f.Title
		if title == "" {
			title = filename
		}
		specs = append(specs, fileSpec{ID: step1.FileID, Title: title})
	}

	filesJSON, err := json.Marshal(specs)
	if err != nil {
		return nil, fmt.Errorf("UploadFiles: marshal files spec: %w", err)
	}
	resp, err := FilesCompleteUploadExternal(ctx, c, &FilesCompleteUploadExternalParams{
		Files:          string(filesJSON),
		ChannelID:      params.ChannelID,
		InitialComment: params.InitialComment,
		ThreadTS:       params.ThreadTS,
	})
	if err != nil {
		return nil, fmt.Errorf("files.completeUploadExternal: %w", err)
	}
	if len(resp.Files) != len(specs) {
		return nil, fmt.Errorf("files.completeUploadExternal: shared %d of %d files", len(resp.Files), len(specs))
	}
	return resp, nil
}

// putFileToPresignedURL streams the file to Slack's presigned URL. The URL
// is unauthenticated (Slack rejects the bearer token there), and the request
// is bounded by ctx, not by c.HTTP's per-request timeout.
func putFileToPresignedURL(ctx context.Context, c *Client, uploadURL, filePath, filename string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("UploadFiles: open %s: %w", filePath, err)
	}
	defer f.Close()

	body, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	go func() {
		part, err := form.CreateFormFile("filename", filename)
		if err == nil {
			_, err = io.Copy(part, f)
		}
		if err == nil {
			err = form.Close()
		}
		writer.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		body.Close()
		return err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())

	httpClient := &http.Client{Transport: c.HTTP.Transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload to presigned URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("upload to presigned URL: HTTP %d (%s)", resp.StatusCode, truncate(string(respBody), 200))
	}
	return nil
}
