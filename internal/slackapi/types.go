//go:generate go run ../../cmd/gen-api -spec ../../api/specs/slack_web.json -out-dir .

// Hand-written augmentations to types.gen.go.
//
// Most types come from the OpenAPI spec via cmd/gen-api. This file is
// just for things the spec doesn't cover:
//
//   - Methods missing from Slack's published spec (e.g. search.files)
//   - Convenience helpers (HistoryPayload) that compose generated types
//
// If you find yourself adding a type that mirrors a spec definition,
// stop — extend the generator instead so the spec stays the source of
// truth.
package slackapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

func strconvItoa(n int) string { return strconv.Itoa(n) }

// HistoryPayload is the typed shape of a conversations.history /
// conversations.replies response body — used by callers that just want
// the messages without touching .Raw.
type HistoryPayload struct {
	Messages []*Message `json:"messages"`
	HasMore  bool       `json:"has_more,omitempty"`
}

// ─── Augmentations for methods missing from Slack's published OpenAPI ─────
//
// search.files is live in Slack's API but absent from
// slack_web_openapi_v2.json. Hand-modelled here until they fix the spec.

// SearchFilesParams mirrors SearchMessagesParams for the search.files method.
type SearchFilesParams struct {
	Count     int    `url:"count,omitempty"`
	Highlight bool   `url:"highlight,omitempty"`
	Page      int    `url:"page,omitempty"`
	Query     string `url:"query"`
	Sort      string `url:"sort,omitempty"`
	SortDir   string `url:"sort_dir,omitempty"`
}

func (p *SearchFilesParams) values() url.Values {
	v := url.Values{}
	if p == nil {
		return v
	}
	if p.Count != 0 {
		v.Set("count", strconvItoa(p.Count))
	}
	if p.Highlight {
		v.Set("highlight", "true")
	}
	if p.Page != 0 {
		v.Set("page", strconvItoa(p.Page))
	}
	v.Set("query", p.Query)
	if p.Sort != "" {
		v.Set("sort", p.Sort)
	}
	if p.SortDir != "" {
		v.Set("sort_dir", p.SortDir)
	}
	return v
}

// SearchFilesResponse is the typed response envelope for search.files.
type SearchFilesResponse struct {
	BaseResponse
	Raw json.RawMessage `json:"-"`
}

func (r *SearchFilesResponse) envelope() *BaseResponse { return &r.BaseResponse }
