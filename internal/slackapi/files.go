// Hand-written file-upload helper.
//
// Slack's files.upload endpoint takes multipart/form-data with the file
// content as a binary part. The generated FilesUpload wrapper assumes
// url-encoded form like every other Slack method, which only works for
// short text content. This file does the proper multipart dance so any
// binary file (PDFs, images, etc.) uploads correctly.
//
// Newer code should target files.getUploadURLExternal + files.completeUploadExternal,
// but the legacy files.upload still works. Switch when Slack actually
// retires the endpoint.
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
)

// UploadFileParams is the option set for UploadFile.
type UploadFileParams struct {
	FilePath       string // path on disk; required
	Filename       string // override filename (defaults to basename of FilePath)
	Title          string
	InitialComment string
	Channels       string // comma-separated channel IDs/names
	ThreadTS       string
	Filetype       string // optional Slack filetype identifier
}

// UploadFileResponse is the typed response envelope for files.upload.
type UploadFileResponse struct {
	BaseResponse
	Raw json.RawMessage `json:"-"`
	// File holds the parsed top-level "file" object — common-case fields only.
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

// UploadFile uploads a file via Slack's legacy files.upload endpoint using
// proper multipart/form-data. Intended for binary files where the generated
// FilesUpload wrapper's url-encoded form won't work.
func UploadFile(ctx context.Context, c *Client, params *UploadFileParams) (*UploadFileResponse, error) {
	if params == nil || params.FilePath == "" {
		return nil, fmt.Errorf("UploadFile: FilePath is required")
	}

	f, err := os.Open(params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("UploadFile: open %s: %w", params.FilePath, err)
	}
	defer f.Close()

	filename := params.Filename
	if filename == "" {
		filename = filepath.Base(params.FilePath)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Helper for writing string fields when they're set.
	addField := func(name, value string) error {
		if value == "" {
			return nil
		}
		return writer.WriteField(name, value)
	}
	if err := addField("filename", filename); err != nil {
		return nil, err
	}
	if err := addField("title", params.Title); err != nil {
		return nil, err
	}
	if err := addField("initial_comment", params.InitialComment); err != nil {
		return nil, err
	}
	if err := addField("channels", params.Channels); err != nil {
		return nil, err
	}
	if err := addField("thread_ts", params.ThreadTS); err != nil {
		return nil, err
	}
	if err := addField("filetype", params.Filetype); err != nil {
		return nil, err
	}

	// File field — must come after string fields.
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("UploadFile: create form file: %w", err)
	}
	if _, err := io.Copy(fileWriter, f); err != nil {
		return nil, fmt.Errorf("UploadFile: copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := c.BaseURL + "/files.upload"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("files.upload: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("files.upload: read body: %w", err)
	}

	var out UploadFileResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return &out, fmt.Errorf("files.upload: decode response: %w (body: %s)", err, truncate(string(raw), 200))
	}
	out.Raw = raw

	if !out.Ok {
		return &out, &APIError{Method: "files.upload", Code: out.Error, Warning: out.Warning}
	}

	return &out, nil
}
