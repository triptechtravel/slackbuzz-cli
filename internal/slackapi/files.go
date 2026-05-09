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
// The public UploadFile signature is unchanged so call sites don't need
// to touch anything; the 3-step dance is internal.
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

	info, err := os.Stat(params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("UploadFile: stat %s: %w", params.FilePath, err)
	}
	size := info.Size()

	filename := params.Filename
	if filename == "" {
		filename = filepath.Base(params.FilePath)
	}

	// Step 1: getUploadURLExternal — get presigned URL + file_id
	step1Form := url.Values{}
	step1Form.Set("filename", filename)
	step1Form.Set("length", strconv.FormatInt(size, 10))
	if params.AltTxt != "" {
		step1Form.Set("alt_txt", params.AltTxt)
	}
	if params.SnippetType != "" {
		step1Form.Set("snippet_type", params.SnippetType)
	}

	var step1 uploadStep1Response
	if _, err := Do(ctx, c, "files.getUploadURLExternal", step1Form, &step1); err != nil {
		return nil, fmt.Errorf("files.getUploadURLExternal: %w", err)
	}
	if step1.UploadURL == "" || step1.FileID == "" {
		return nil, fmt.Errorf("files.getUploadURLExternal: empty upload_url or file_id in response")
	}

	// Step 2: POST file content to the presigned URL via multipart/form-data
	if err := putFileToPresignedURL(ctx, c, step1.UploadURL, params.FilePath, filename); err != nil {
		return nil, err
	}

	// Step 3: completeUploadExternal — share the file
	type fileSpec struct {
		ID    string `json:"id"`
		Title string `json:"title,omitempty"`
	}
	title := params.Title
	if title == "" {
		title = filename
	}
	filesJSON, err := json.Marshal([]fileSpec{{ID: step1.FileID, Title: title}})
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

	out := &UploadFileResponse{
		BaseResponse: step3.BaseResponse,
		Raw:          raw,
	}
	if len(step3.Files) > 0 {
		f := step3.Files[0]
		out.File.ID = f.ID
		out.File.Title = f.Title
		out.File.Name = f.Name
		out.File.Mimetype = f.Mimetype
		out.File.URL = f.URL
		out.File.Permalink = f.Permalink
	} else {
		// Fall back to the file_id from step 1 so callers always have
		// something to reference.
		out.File.ID = step1.FileID
		out.File.Title = title
		out.File.Name = filename
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
