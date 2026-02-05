package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsMissingScopeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"missing_scope direct", fmt.Errorf("missing_scope"), true},
		{"wrapped missing_scope", fmt.Errorf("lookup failed: %w", fmt.Errorf("missing_scope")), true},
		{"contains missing_scope", fmt.Errorf("error: missing_scope for channels:read"), true},
		{"unrelated error", fmt.Errorf("channel_not_found"), false},
		{"empty error", fmt.Errorf(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMissingScopeError(tt.err)
			if got != tt.want {
				t.Errorf("IsMissingScopeError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFormatResolveError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target string
		check  func(string) bool
	}{
		{
			name:   "nil error",
			err:    nil,
			target: "#general",
			check:  func(s string) bool { return s == "" },
		},
		{
			name:   "missing_scope error mentions target",
			err:    fmt.Errorf("missing_scope"),
			target: "#general",
			check: func(s string) bool {
				return strings.Contains(s, "#general") && strings.Contains(s, "missing_scope")
			},
		},
		{
			name:   "missing_scope suggests app create",
			err:    fmt.Errorf("missing_scope"),
			target: "#dev",
			check: func(s string) bool {
				return strings.Contains(s, "slackbuzz app create")
			},
		},
		{
			name:   "non-scope error includes target",
			err:    fmt.Errorf("channel_not_found"),
			target: "#nonexistent",
			check: func(s string) bool {
				return strings.Contains(s, "#nonexistent") && strings.Contains(s, "channel_not_found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatResolveError(tt.err, tt.target)
			if !tt.check(got) {
				t.Errorf("FormatResolveError() = %q, check failed", got)
			}
		})
	}
}

func TestFormatSendError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target string
		check  func(string) bool
	}{
		{
			name:   "missing_scope for DM suggests im:write",
			err:    fmt.Errorf("missing_scope"),
			target: "@alice",
			check: func(s string) bool {
				return strings.Contains(s, "im:write")
			},
		},
		{
			name:   "missing_scope for channel suggests chat:write",
			err:    fmt.Errorf("missing_scope"),
			target: "#general",
			check: func(s string) bool {
				return strings.Contains(s, "chat:write")
			},
		},
		{
			name:   "non-scope error uses friendly message",
			err:    fmt.Errorf("channel_not_found"),
			target: "#gone",
			check: func(s string) bool {
				return strings.Contains(s, "Channel not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSendError(tt.err, tt.target)
			if !tt.check(got) {
				t.Errorf("FormatSendError() = %q, check failed", got)
			}
		})
	}
}
