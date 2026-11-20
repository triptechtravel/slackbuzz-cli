// Hand-written file-upload helper.
//
// Slack hard-deprecated /files.upload (it returns method_deprecated). Its
// replacement is a 3-step external flow; steps 1 and 3 are ordinary API
// methods with generated wrappers (added to the spec in cmd/gen-api/patches.go,
// since Slack's published spec predates them):
//
//  1. FilesGetUploadURLExternal — filename + length → presigned URL + file_id.
//  2. POST the bytes to the presigned URL (multipart, "filename" field). Not a
//     Slack API method and unauthenticated: the only hand-written step.
//  3. FilesCompleteUploadExternal — shares every file_id in ONE call, so all
//     the files arrive as one message with the initial comment.
package slackapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadFileSpec is one file in an UploadFiles call.
type UploadFileSpec struct {
	FilePath    string // path on disk; required
	Filename    string // defaults to basename of FilePath
	Title       string // defaults to Filename
	AltTxt      string
	SnippetType string
}

// UploadFilesParams shares several files as one message.
type UploadFilesParams struct {
	Files          []UploadFileSpec
	Channels       string // single channel_id; if comma-separated, the first wins
	InitialComment string
	ThreadTS       string
}

// UploadFiles uploads every file (steps 1-2), then shares them together in a
// single completeUploadExternal call: one message carrying all the files and
// the initial comment.
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
		filename := f.Filename
		if filename == "" {
			filename = filepath.Base(f.FilePath)
		}

		step1, err := FilesGetUploadURLExternal(ctx, c, &FilesGetUploadURLExternalParams{
			Filename: filename, Length: int(info.Size()), AltTxt: f.AltTxt, SnippetType: f.SnippetType,
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
	// The endpoint takes a single channel_id; if a caller passed several, the first wins.
	channel, _, _ := strings.Cut(params.Channels, ",")
	resp, err := FilesCompleteUploadExternal(ctx, c, &FilesCompleteUploadExternalParams{
		Files:          string(filesJSON),
		ChannelID:      strings.TrimSpace(channel),
		InitialComment: params.InitialComment,
		ThreadTS:       params.ThreadTS,
	})
	if err != nil {
		return nil, fmt.Errorf("files.completeUploadExternal: %w", err)
	}
	return resp, nil
}

// putFileToPresignedURL uploads the file content to Slack's presigned
// URL. The presigned URL is unauthenticated — Slack rejects requests
// that send the bearer token to it.
func putFileToPresignedURL(ctx context.Context, c *Client, uploadURL, filePath, filename string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("UploadFiles: open %s: %w", filePath, err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("filename", filename)
	if err != nil {
		return fmt.Errorf("UploadFiles: create form file: %w", err)
	}
	if _, err := io.Copy(fileWriter, f); err != nil {
		return fmt.Errorf("UploadFiles: copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTP.Do(req)
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
