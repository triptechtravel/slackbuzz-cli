// Hand-written file-upload helper.
//
// Slack hard-deprecated the legacy /files.upload endpoint in 2026 and
// fresh requests now return method_deprecated. Modern uploads use a
// 3-step external flow:
//
//  1. files.getUploadURLExternal — POST filename + length, get back a
//     presigned upload URL and a file_id.
//  2. POST the file content to the presigned URL (multipart/form-data,
//     "filename" field). No auth header on this hop.
//  3. files.completeUploadExternal — POST the file_id (in a JSON-string
//     array along with optional title), channel_id, initial_comment,
//     thread_ts. This is the call that actually shares the file.
//
// UploadFiles runs steps 1-2 per file and step 3 once, so several files
// arrive as ONE message (with the initial comment), the way the Slack client
// shares them. UploadFile is the one-file case.
package slackapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// UploadFileParams is the option set for UploadFile.
type UploadFileParams struct {
	FilePath       string // path on disk; required
	Filename       string // override filename (defaults to basename of FilePath)
	Title          string
	InitialComment string
	Channels       string // single channel_id; if comma-separated, the first wins
	ThreadTS       string
	AltTxt         string // accessibility text (images)
	SnippetType    string // for text snippets, e.g. "text", "go"
}

// UploadFileResponse is the typed response envelope for the upload flow.
// Shape preserved from the legacy files.upload response so callers don't
// change — we synthesise the .File fields from completeUploadExternal's
// `files[0]` array entry.
type UploadFileResponse struct {
	BaseResponse
	Raw  json.RawMessage `json:"-"`
	File struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Name      string `json:"name"`
		Mimetype  string `json:"mimetype"`
		URL       string `json:"url_private"`
		Permalink string `json:"permalink"`
	} `json:"file"`
}

func (r *UploadFileResponse) envelope() *BaseResponse { return &r.BaseResponse }

// uploadStep1Response is the typed response from files.getUploadURLExternal.
type uploadStep1Response struct {
	BaseResponse
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
}

func (r *uploadStep1Response) envelope() *BaseResponse { return &r.BaseResponse }

// uploadStep3File mirrors a single entry in completeUploadExternal's
// `files` response array.
type uploadStep3File struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Name      string `json:"name"`
	Mimetype  string `json:"mimetype"`
	URL       string `json:"url_private"`
	Permalink string `json:"permalink"`
}

// uploadStep3Response is the typed response from files.completeUploadExternal.
type uploadStep3Response struct {
	BaseResponse
	Files []uploadStep3File `json:"files"`
}

func (r *uploadStep3Response) envelope() *BaseResponse { return &r.BaseResponse }

// UploadFile uploads a file via Slack's modern external-upload flow
// (files.getUploadURLExternal → presigned PUT → files.completeUploadExternal).
// Replaces the deprecated files.upload endpoint.
func UploadFile(ctx context.Context, c *Client, params *UploadFileParams) (*UploadFileResponse, error) {
	if params == nil || params.FilePath == "" {
		return nil, fmt.Errorf("UploadFile: FilePath is required")
	}
	resp, err := UploadFiles(ctx, c, &UploadFilesParams{
		Files:          []UploadFileSpec{{FilePath: params.FilePath, Filename: params.Filename, Title: params.Title, AltTxt: params.AltTxt, SnippetType: params.SnippetType}},
		Channels:       params.Channels,
		InitialComment: params.InitialComment,
		ThreadTS:       params.ThreadTS,
	})
	if err != nil {
		return nil, err
	}
	out := &UploadFileResponse{BaseResponse: resp.BaseResponse, Raw: resp.Raw}
	f := resp.Files[0]
	out.File.ID, out.File.Title, out.File.Name = f.ID, f.Title, f.Name
	out.File.Mimetype, out.File.URL, out.File.Permalink = f.Mimetype, f.URL, f.Permalink
	return out, nil
}

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

// UploadedFile is one shared file, in the order the files were given.
type UploadedFile = uploadStep3File

// UploadFilesResponse is the outcome of one UploadFiles call.
type UploadFilesResponse struct {
	BaseResponse
	Raw   json.RawMessage
	Files []UploadedFile
}

// UploadFiles uploads every file (steps 1-2), then shares them together in a
// single files.completeUploadExternal call: one message carrying all the
// files and the initial comment. Files come back in the order given.
func UploadFiles(ctx context.Context, c *Client, params *UploadFilesParams) (*UploadFilesResponse, error) {
	if params == nil || len(params.Files) == 0 {
		return nil, fmt.Errorf("UploadFiles: at least one file is required")
	}
	type fileSpec struct {
		ID    string `json:"id"`
		Title string `json:"title,omitempty"`
	}
	specs := make([]fileSpec, 0, len(params.Files))
	fallback := make([]UploadedFile, 0, len(params.Files))
	for _, f := range params.Files {
		if f.FilePath == "" {
			return nil, fmt.Errorf("UploadFile: FilePath is required")
		}
		info, err := os.Stat(f.FilePath)
		if err != nil {
			return nil, fmt.Errorf("UploadFile: stat %s: %w", f.FilePath, err)
		}
		filename := f.Filename
		if filename == "" {
			filename = filepath.Base(f.FilePath)
		}

		// Step 1: getUploadURLExternal — get presigned URL + file_id
		step1Form := url.Values{}
		step1Form.Set("filename", filename)
		step1Form.Set("length", strconv.FormatInt(info.Size(), 10))
		if f.AltTxt != "" {
			step1Form.Set("alt_txt", f.AltTxt)
		}
		if f.SnippetType != "" {
			step1Form.Set("snippet_type", f.SnippetType)
		}
		var step1 uploadStep1Response
		if _, err := Do(ctx, c, "files.getUploadURLExternal", step1Form, &step1); err != nil {
			return nil, fmt.Errorf("files.getUploadURLExternal: %w", err)
		}
		if step1.UploadURL == "" || step1.FileID == "" {
			return nil, fmt.Errorf("files.getUploadURLExternal: empty upload_url or file_id in response")
		}

		// Step 2: POST file content to the presigned URL via multipart/form-data
		if err := putFileToPresignedURL(ctx, c, step1.UploadURL, f.FilePath, filename); err != nil {
			return nil, err
		}

		title := f.Title
		if title == "" {
			title = filename
		}
		specs = append(specs, fileSpec{ID: step1.FileID, Title: title})
		fallback = append(fallback, UploadedFile{ID: step1.FileID, Title: title, Name: filename})
	}

	// Step 3: completeUploadExternal — share every file in one message
	filesJSON, err := json.Marshal(specs)
	if err != nil {
		return nil, fmt.Errorf("UploadFile: marshal files spec: %w", err)
	}

	step3Form := url.Values{}
	step3Form.Set("files", string(filesJSON))
	if params.Channels != "" {
		// Slack's new endpoint takes a single channel_id. If a caller passed
		// multiple comma-separated, take the first.
		first := params.Channels
		if i := strings.IndexByte(first, ','); i >= 0 {
			first = first[:i]
		}
		step3Form.Set("channel_id", strings.TrimSpace(first))
	}
	if params.InitialComment != "" {
		step3Form.Set("initial_comment", params.InitialComment)
	}
	if params.ThreadTS != "" {
		step3Form.Set("thread_ts", params.ThreadTS)
	}

	var step3 uploadStep3Response
	raw, err := Do(ctx, c, "files.completeUploadExternal", step3Form, &step3)
	if err != nil {
		return nil, fmt.Errorf("files.completeUploadExternal: %w", err)
	}
	// Slack echoes each shared file; without that, fall back to step 1's ids
	// so callers always have something to reference.
	out := &UploadFilesResponse{BaseResponse: step3.BaseResponse, Raw: raw, Files: fallback}
	if len(step3.Files) == len(fallback) {
		out.Files = step3.Files
	}
	return out, nil
}

// putFileToPresignedURL uploads the file content to Slack's presigned
// URL. The presigned URL is unauthenticated — Slack rejects requests
// that send the bearer token to it.
func putFileToPresignedURL(ctx context.Context, c *Client, uploadURL, filePath, filename string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("UploadFile: open %s: %w", filePath, err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("filename", filename)
	if err != nil {
		return fmt.Errorf("UploadFile: create form file: %w", err)
	}
	if _, err := io.Copy(fileWriter, f); err != nil {
		return fmt.Errorf("UploadFile: copy file content: %w", err)
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
