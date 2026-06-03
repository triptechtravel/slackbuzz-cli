package slackapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServer spins up an httptest server pointed at the given handler and
// returns a Client whose BaseURL targets it. Mirrors the clickup-cli
// pattern in `internal/apiv2/operations_test.go`.
func testServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New("xoxp-test-token")
	c.BaseURL = srv.URL
	return srv, c
}

// ---------------------------------------------------------------------------
// Transport (Do) — common to every generated wrapper
// ---------------------------------------------------------------------------

func TestDo_AuthHeaderAndForm(t *testing.T) {
	var capturedAuth, capturedCT, capturedBody string
	var capturedPath string

	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedCT = r.Header.Get("Content-Type")
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, err := AuthTest(context.Background(), c)
	require.NoError(t, err)

	assert.Equal(t, "Bearer xoxp-test-token", capturedAuth)
	assert.Equal(t, "application/x-www-form-urlencoded", capturedCT)
	assert.Equal(t, "/auth.test", capturedPath)
	assert.Empty(t, capturedBody, "auth.test should send no params")
}

func TestDo_FormParamsAreEncoded(t *testing.T) {
	var capturedForm url.Values

	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, err := ConversationsHistory(context.Background(), c, &ConversationsHistoryParams{
		Channel: "C123",
		Limit:   50,
	})
	require.NoError(t, err)

	assert.Equal(t, "C123", capturedForm.Get("channel"))
	assert.Equal(t, "50", capturedForm.Get("limit"))
}

func TestDo_OmitsZeroValueOptionalParams(t *testing.T) {
	var capturedForm url.Values

	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, err := ConversationsHistory(context.Background(), c, &ConversationsHistoryParams{
		Channel: "C123",
		// Limit, Latest etc. left at zero — should be omitted entirely.
	})
	require.NoError(t, err)

	assert.NotContains(t, capturedForm, "limit", "zero-value optional fields must not be sent")
	assert.NotContains(t, capturedForm, "latest")
	assert.NotContains(t, capturedForm, "oldest")
}

// ---------------------------------------------------------------------------
// Response handling
// ---------------------------------------------------------------------------

func TestResponse_OkAndRawBody(t *testing.T) {
	const body = `{"ok":true,"user":"isaac","team":"Triptech","team_id":"T0CPKU0VA"}`

	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	resp, err := AuthTest(context.Background(), c)
	require.NoError(t, err)
	assert.True(t, resp.Ok)
	assert.JSONEq(t, body, string(resp.Raw))

	// Caller-side typed access via Raw — the canonical pattern.
	var info struct {
		User string `json:"user"`
		Team string `json:"team"`
	}
	require.NoError(t, json.Unmarshal(resp.Raw, &info))
	assert.Equal(t, "isaac", info.User)
	assert.Equal(t, "Triptech", info.Team)
}

func TestResponse_NotOkReturnsAPIError(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	})

	_, err := ConversationsHistory(context.Background(), c, &ConversationsHistoryParams{Channel: "Cdoesnotexist"})
	require.Error(t, err)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr), "should be *APIError")
	assert.Equal(t, "channel_not_found", apiErr.Code)
	assert.Equal(t, "conversations.history", apiErr.Method)

	// Sentinel errors.Is matching — the canonical caller-side pattern.
	assert.True(t, errors.Is(err, ErrChannelNotFound))
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestResponse_MissingScopeExtractsScopes(t *testing.T) {
	// Slack's missing_scope envelope returns top-level `needed`/`provided`
	// fields outside the standard envelope. The transport must surface
	// them as typed APIError fields so callers (and `slackbuzz doctor`)
	// can suggest specific scope additions.
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"missing_scope","needed":"im:history,mpim:history","provided":"identify,users:read"}`))
	})

	_, err := ConversationsHistory(context.Background(), c, &ConversationsHistoryParams{Channel: "D123"})
	require.Error(t, err)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "missing_scope", apiErr.Code)
	assert.Equal(t, []string{"im:history", "mpim:history"}, apiErr.NeededScopes)
	assert.Equal(t, []string{"identify", "users:read"}, apiErr.ProvidedScopes)
}

func TestResponse_DecodeFailureWrapsRawBody(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`<html>not json</html>`))
	})

	_, err := AuthTest(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
	assert.Contains(t, err.Error(), "not json", "raw body should appear in the error for debugging")
}

// ---------------------------------------------------------------------------
// Scope map
// ---------------------------------------------------------------------------

func TestMethodScopes_KnownMethods(t *testing.T) {
	tests := []struct {
		method string
		want   []string // any of these must appear
	}{
		{"conversations.history", []string{"channels:history", "groups:history", "im:history", "mpim:history"}},
		// Slack splits chat:write into :bot and :user variants in the spec —
		// either qualifies depending on which token type makes the call.
		{"chat.postMessage", []string{"chat:write:bot"}},
		{"reactions.add", []string{"reactions:write"}},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got, ok := MethodScopes[tt.method]
			require.True(t, ok, "expected method %s in MethodScopes", tt.method)
			for _, want := range tt.want {
				assert.Contains(t, got, want, "%s should require %s", tt.method, want)
			}
		})
	}
}

func TestScopesForMethods_Union(t *testing.T) {
	got := ScopesForMethods([]string{"conversations.history", "conversations.list"})
	// Should contain history scopes.
	assert.Contains(t, got, "channels:history")
	assert.Contains(t, got, "im:history")
	// And read scopes from list.
	assert.Contains(t, got, "channels:read")
}

func TestScopesForMethods_Deduplicates(t *testing.T) {
	got := ScopesForMethods([]string{"conversations.history", "conversations.history"})
	seen := map[string]int{}
	for _, s := range got {
		seen[s]++
	}
	for s, n := range seen {
		assert.Equal(t, 1, n, "scope %s appeared %d times", s, n)
	}
}

// ---------------------------------------------------------------------------
// Param encoding edge cases
// ---------------------------------------------------------------------------

func TestParams_BoolEncoding(t *testing.T) {
	var captured url.Values
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, err := ConversationsHistory(context.Background(), c, &ConversationsHistoryParams{
		Channel:   "C123",
		Inclusive: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "true", captured.Get("inclusive"))
}

func TestParams_NilSafe(t *testing.T) {
	// Calling with nil params should not panic and should send no form data
	// beyond what the method requires.
	var captured string
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, err := ConversationsHistory(context.Background(), c, nil)
	require.NoError(t, err)
	assert.True(t, strings.TrimSpace(captured) == "" || !strings.Contains(captured, "channel="))
}
