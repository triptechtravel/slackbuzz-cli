// Hand-written operation wrappers for Slack methods that are live in the
// API but missing from slack_web_openapi_v2.json. Keep this small —
// regenerate from the spec instead whenever Slack ships an updated one.
package slackapi

import (
	"context"
)

// SearchFiles calls Slack's search.files method.
//
// The Slack Web API supports this method but their published OpenAPI 2.0
// spec doesn't list it. Hand-written here until they fix the spec.
//
// Required scopes: search:read.
func SearchFiles(ctx context.Context, c *Client, params *SearchFilesParams) (*SearchFilesResponse, error) {
	var resp SearchFilesResponse
	form := params.values()
	body, err := Do(ctx, c, "search.files", form, &resp)
	resp.Raw = body
	if err != nil {
		return &resp, err
	}
	return &resp, nil
}
