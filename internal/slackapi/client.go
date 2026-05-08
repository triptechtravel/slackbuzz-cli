// Package slackapi is the typed Slack Web API client used by slackbuzz-cli.
//
// It mirrors the clickup-cli `internal/apiv{2,3}` pattern: a small hand-written
// transport (this file) plus generated operation wrappers + types
// (operations.gen.go, types.gen.go, scopes.gen.go) produced from
// `api/specs/slack_web.json` via `cmd/gen-api`.
//
// The hand-written surface is intentionally minimal — everything that
// could drift from the spec is generated. Add an operation? Run `make
// api-gen`. Add a scope? It comes for free from the spec.
package slackapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BaseURL is Slack's Web API root.
const BaseURL = "https://slack.com/api"

// Client is the minimal transport for Slack Web API calls.
//
// Slack's Web API is a single-host, form-encoded, JSON-response service —
// roughly: POST https://slack.com/api/<method> with a bearer token and
// either form params or a JSON body. We don't need the full ceremony of
// REST routing.
type Client struct {
	HTTP    *http.Client
	Token   string
	BaseURL string // for tests; defaults to BaseURL
}

// New constructs a Client with the given OAuth token and a sensible HTTP
// timeout. Use NewWithHTTP when you need to override the transport (tests,
// custom retry policy, etc).
func New(token string) *Client {
	return NewWithHTTP(token, &http.Client{Timeout: 30 * time.Second})
}

// NewWithHTTP constructs a Client with a caller-supplied http.Client.
func NewWithHTTP(token string, hc *http.Client) *Client {
	return &Client{HTTP: hc, Token: token, BaseURL: BaseURL}
}

// BaseResponse captures the envelope every Slack Web API call returns.
// All generated *Response types embed it.
type BaseResponse struct {
	Ok               bool   `json:"ok"`
	Error            string `json:"error,omitempty"`
	Warning          string `json:"warning,omitempty"`
	ResponseMetadata struct {
		Messages   []string `json:"messages,omitempty"`
		NextCursor string   `json:"next_cursor,omitempty"`
	} `json:"response_metadata,omitempty"`
}

// AsError returns a non-nil error if the response indicates failure.
func (b *BaseResponse) AsError(method string) error {
	if b.Ok {
		return nil
	}
	if b.Error == "" {
		return fmt.Errorf("%s: api returned ok=false with no error message", method)
	}
	return &APIError{Method: method, Code: b.Error, Warning: b.Warning}
}

// Slack's error code constants. Use errors.Is(err, ErrMissingScope) etc.
// to test for specific failure modes — see the sentinel errors below.
const (
	CodeMissingScope      = "missing_scope"
	CodeNotAuthed         = "not_authed"
	CodeInvalidAuth       = "invalid_auth"
	CodeAccountInactive   = "account_inactive"
	CodeTokenRevoked      = "token_revoked"
	CodeRatelimited       = "ratelimited"
	CodeChannelNotFound   = "channel_not_found"
	CodeUserNotFound      = "user_not_found"
	CodeNotInChannel      = "not_in_channel"
	CodeIsArchived        = "is_archived"
	CodeMessageNotFound   = "message_not_found"
	CodeCantUpdateMessage = "cant_update_message"
	CodeCantDeleteMessage = "cant_delete_message"
)

// Sentinel errors callers can match against with errors.Is.
//
// Each sentinel corresponds to one of the well-known Slack error codes
// listed above. The transport returns an *APIError; errors.Is walks its
// Unwrap chain to match these sentinels by code.
var (
	ErrMissingScope    = &APIError{Code: CodeMissingScope}
	ErrNotAuthed       = &APIError{Code: CodeNotAuthed}
	ErrInvalidAuth     = &APIError{Code: CodeInvalidAuth}
	ErrAccountInactive = &APIError{Code: CodeAccountInactive}
	ErrTokenRevoked    = &APIError{Code: CodeTokenRevoked}
	ErrRatelimited     = &APIError{Code: CodeRatelimited}
	ErrChannelNotFound = &APIError{Code: CodeChannelNotFound}
	ErrUserNotFound    = &APIError{Code: CodeUserNotFound}
	ErrNotInChannel    = &APIError{Code: CodeNotInChannel}
	ErrIsArchived      = &APIError{Code: CodeIsArchived}
	ErrMessageNotFound = &APIError{Code: CodeMessageNotFound}
)

// APIError is the typed error returned by every operation when Slack's
// response envelope says ok=false.
//
// Match specific failures with errors.Is:
//
//	if errors.Is(err, slackapi.ErrChannelNotFound) {
//	    // …handle the user typing a stale channel ID
//	}
//
// Or pull the typed value to read scope context:
//
//	var apiErr *slackapi.APIError
//	if errors.As(err, &apiErr) && apiErr.Code == slackapi.CodeMissingScope {
//	    fmt.Printf("Missing: %v\n", apiErr.NeededScopes)
//	}
type APIError struct {
	Method  string
	Code    string // Slack's machine-readable error string ("missing_scope", "channel_not_found", ...)
	Warning string
	// NeededScopes / ProvidedScopes are populated by the transport when
	// the response carries Slack's `needed`/`provided` scope split. Only
	// applies to Code == CodeMissingScope.
	NeededScopes   []string
	ProvidedScopes []string
}

func (e *APIError) Error() string {
	if len(e.NeededScopes) > 0 {
		return fmt.Sprintf("%s: %s (needed scopes: %s)", e.Method, e.Code, strings.Join(e.NeededScopes, ", "))
	}
	if e.Method != "" {
		return fmt.Sprintf("%s: %s", e.Method, e.Code)
	}
	return e.Code
}

// Is supports errors.Is matching against the sentinel error values
// declared above. The match is by Code only — Method/Warning are ignored.
func (e *APIError) Is(target error) bool {
	other, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.Code == other.Code
}

// Envelope is implemented by every generated *Response type. It exposes
// the embedded BaseResponse so the transport can check ok/error
// uniformly without reflection.
type Envelope interface {
	envelope() *BaseResponse
}

// Do POSTs the given form values to the named Slack method, decodes the
// JSON response into out (which must implement Envelope, normally by
// embedding BaseResponse), and returns the raw response body for callers
// who need typed access to method-specific fields.
//
// Generated wrappers all bottom out here. Callers do not normally invoke
// Do directly — use ConversationsHistory(...) etc.
func Do(ctx context.Context, c *Client, method string, form url.Values, out Envelope) ([]byte, error) {
	endpoint := c.BaseURL + "/" + method

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", method, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("%s: read body: %w", method, err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return body, fmt.Errorf("%s: decode response: %w (body: %s)", method, err, truncate(string(body), 200))
	}

	env := out.envelope()
	if env.Ok {
		return body, nil
	}

	// On missing_scope, Slack also returns top-level `needed`/`provided`
	// fields outside the envelope. Re-decode raw to pull them out for the
	// typed APIError so callers can suggest scope fixes.
	if env.Error == "missing_scope" {
		var raw struct {
			Needed   string `json:"needed"`
			Provided string `json:"provided"`
		}
		_ = json.Unmarshal(body, &raw)
		return body, &APIError{
			Method:         method,
			Code:           env.Error,
			Warning:        env.Warning,
			NeededScopes:   splitScopes(raw.Needed),
			ProvidedScopes: splitScopes(raw.Provided),
		}
	}

	return body, env.AsError(method)
}

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
