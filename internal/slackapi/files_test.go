package slackapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadTestServer drives the three-step files external-upload flow:
//
//	POST /api/files.getUploadURLExternal  → returns a presigned upload URL
//	POST /upload-presigned/<file_id>      → accepts the file body
//	POST /api/files.completeUploadExternal → finalises the upload
//
// All three hit the same httptest server but on different paths, so the
// test can observe every leg of the flow.
type uploadTestRecord struct {
	step1Form      url.Values
	uploadBody     []byte
	uploadFilename string
	step3Form      url.Values
}

func uploadTestServer(t *testing.T) (*Client, *uploadTestRecord) {
	t.Helper()
	rec := &uploadTestRecord{}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		rec.step1Form = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + srv.URL + `/upload-presigned/F123","file_id":"F123"}`))
	})

	mux.HandleFunc("/upload-presigned/F123", func(w http.ResponseWriter, r *http.Request) {
		// Slack's presigned URL is unauthenticated — fail loudly if a
		// caller leaks the bearer token to it.
		assert.Empty(t, r.Header.Get("Authorization"), "presigned URL must not receive bearer token")

		ct := r.Header.Get("Content-Type")
		require.True(t, strings.HasPrefix(ct, "multipart/form-data"), "expected multipart, got %q", ct)
		_, params, err := mime.ParseMediaType(ct)
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			if part.FormName() == "filename" {
				rec.uploadFilename = part.FileName()
				body, err := io.ReadAll(part)
				require.NoError(t, err)
				rec.uploadBody = body
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/api/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		rec.step3Form = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":[{"id":"F123","title":"hello","name":"hello.txt","mimetype":"text/plain","url_private":"https://files.slack.com/F123","permalink":"https://slack.com/permalink/F123"}]}`))
	})

	c := New("xoxp-test-token")
	c.BaseURL = srv.URL + "/api"
	return c, rec
}

func TestUploadFile_HappyPath(t *testing.T) {
	tmp := writeTempFile(t, "hello.txt", "hello world")

	c, rec := uploadTestServer(t)

	resp, err := UploadFile(context.Background(), c, &UploadFileParams{
		FilePath:       tmp,
		Title:          "hello",
		InitialComment: "first comment",
		Channels:       "C123",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Step 1 carried filename + length.
	assert.Equal(t, "hello.txt", rec.step1Form.Get("filename"))
	assert.Equal(t, "11", rec.step1Form.Get("length"))

	// Step 2 received the file body, no auth header.
	assert.Equal(t, "hello world", string(rec.uploadBody))
	assert.Equal(t, "hello.txt", rec.uploadFilename)

	// Step 3 carried channel_id, initial_comment, and a JSON files array.
	assert.Equal(t, "C123", rec.step3Form.Get("channel_id"))
	assert.Equal(t, "first comment", rec.step3Form.Get("initial_comment"))
	assert.Contains(t, rec.step3Form.Get("files"), `"id":"F123"`)
	assert.Contains(t, rec.step3Form.Get("files"), `"title":"hello"`)

	// Response shape preserved (callers use .File.ID etc.)
	assert.Equal(t, "F123", resp.File.ID)
	assert.Equal(t, "hello", resp.File.Title)
	assert.Equal(t, "hello.txt", resp.File.Name)
	assert.Equal(t, "https://slack.com/permalink/F123", resp.File.Permalink)
}

func TestUploadFile_CommaSeparatedChannelsTakesFirst(t *testing.T) {
	tmp := writeTempFile(t, "x.txt", "hi")
	c, rec := uploadTestServer(t)

	_, err := UploadFile(context.Background(), c, &UploadFileParams{
		FilePath: tmp,
		Channels: "C123,C456,C789",
	})
	require.NoError(t, err)
	assert.Equal(t, "C123", rec.step3Form.Get("channel_id"))
}

func TestUploadFile_Step1ErrorPropagates(t *testing.T) {
	tmp := writeTempFile(t, "x.txt", "hi")
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"missing_scope","needed":"files:write"}`))
	})
	c := New("xoxp-test-token")
	c.BaseURL = srv.URL + "/api"

	_, err := UploadFile(context.Background(), c, &UploadFileParams{FilePath: tmp})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingScope), "expected ErrMissingScope, got %v", err)
}

func TestUploadFile_Step3ErrorPropagates(t *testing.T) {
	tmp := writeTempFile(t, "x.txt", "hi")
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"upload_url":"` + srv.URL + `/upload","file_id":"F999"}`))
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	})
	c := New("xoxp-test-token")
	c.BaseURL = srv.URL + "/api"

	_, err := UploadFile(context.Background(), c, &UploadFileParams{FilePath: tmp, Channels: "Cbad"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrChannelNotFound), "expected ErrChannelNotFound, got %v", err)
}

// ---- helpers ----

func writeTempFile(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}
