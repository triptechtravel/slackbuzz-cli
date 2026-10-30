package main

import (
	"slices"
	"testing"
)

func TestClassifyScopesUserOnly(t *testing.T) {
	bot, user := classifyScopes([]string{"search:read", "stars:read", "chat:write", "users:read"})

	for s := range userOnlyScopes {
		if slices.Contains(bot, s) {
			t.Errorf("bot scopes contain user-only scope %q", s)
		}
	}
	if !slices.Contains(user, "search:read") {
		t.Errorf("user scopes missing search:read: %v", user)
	}
	if !slices.Contains(bot, "chat:write") || !slices.Contains(user, "chat:write") {
		t.Errorf("unqualified scope chat:write should be on both sides: bot=%v user=%v", bot, user)
	}
}
