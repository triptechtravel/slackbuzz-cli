package config

import (
	"testing"
)

func TestAuthConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SLACKBUZZ_CONFIG_DIR", dir)

	ac := &AuthConfig{
		TeamID:      "T12345",
		TeamName:    "Test Team",
		UserID:      "U_HUMAN",
		UserName:    "isaac",
		BotUserID:   "U_BOT",
		BotUserName: "slackbuzz",
	}

	if err := ac.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth() error: %v", err)
	}

	if loaded.TeamID != "T12345" {
		t.Errorf("TeamID = %q, want %q", loaded.TeamID, "T12345")
	}
	if loaded.UserID != "U_HUMAN" {
		t.Errorf("UserID = %q, want %q", loaded.UserID, "U_HUMAN")
	}
	if loaded.UserName != "isaac" {
		t.Errorf("UserName = %q, want %q", loaded.UserName, "isaac")
	}
	if loaded.BotUserID != "U_BOT" {
		t.Errorf("BotUserID = %q, want %q", loaded.BotUserID, "U_BOT")
	}
	if loaded.BotUserName != "slackbuzz" {
		t.Errorf("BotUserName = %q, want %q", loaded.BotUserName, "slackbuzz")
	}
}

func TestAuthConfigClear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SLACKBUZZ_CONFIG_DIR", dir)

	ac := &AuthConfig{
		BotToken:    "xoxb-test",
		UserToken:   "xoxp-test",
		UserID:      "U_HUMAN",
		UserName:    "isaac",
		BotUserID:   "U_BOT",
		BotUserName: "slackbuzz",
	}

	if err := ac.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err := ac.Clear(); err != nil {
		t.Fatalf("Clear() error: %v", err)
	}

	if ac.BotUserID != "" {
		t.Errorf("BotUserID after Clear() = %q, want empty", ac.BotUserID)
	}
	if ac.BotUserName != "" {
		t.Errorf("BotUserName after Clear() = %q, want empty", ac.BotUserName)
	}
	if ac.UserID != "" {
		t.Errorf("UserID after Clear() = %q, want empty", ac.UserID)
	}
}

func TestAuthConfigBotAndUserSeparate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SLACKBUZZ_CONFIG_DIR", dir)

	// Simulate storing bot info
	ac, _ := LoadAuth()
	ac.BotUserID = "U_BOT"
	ac.BotUserName = "mybot"
	if err := ac.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Simulate storing user info (should not affect bot fields)
	ac, _ = LoadAuth()
	ac.UserID = "U_HUMAN"
	ac.UserName = "isaac"
	if err := ac.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Reload and verify both are independent
	ac, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth() error: %v", err)
	}

	if ac.BotUserID != "U_BOT" || ac.BotUserName != "mybot" {
		t.Errorf("bot info = (%q, %q), want (%q, %q)", ac.BotUserID, ac.BotUserName, "U_BOT", "mybot")
	}
	if ac.UserID != "U_HUMAN" || ac.UserName != "isaac" {
		t.Errorf("user info = (%q, %q), want (%q, %q)", ac.UserID, ac.UserName, "U_HUMAN", "isaac")
	}
}
