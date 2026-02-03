package auth

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/zalando/go-keyring"
)

func setConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SLACKBUZZ_CONFIG_DIR", dir)
	return dir
}

func TestStoreAndGetBotToken(t *testing.T) {
	keyring.MockInit()

	token := "xoxb-test-abc123"
	if err := StoreBotToken(token); err != nil {
		t.Fatalf("StoreBotToken() error: %v", err)
	}

	got, err := GetBotToken()
	if err != nil {
		t.Fatalf("GetBotToken() error: %v", err)
	}
	if got != token {
		t.Errorf("GetBotToken() = %q, want %q", got, token)
	}
}

func TestStoreAndGetUserToken(t *testing.T) {
	keyring.MockInit()

	token := "xoxp-test-abc123"
	if err := StoreUserToken(token); err != nil {
		t.Fatalf("StoreUserToken() error: %v", err)
	}

	got, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken() error: %v", err)
	}
	if got != token {
		t.Errorf("GetUserToken() = %q, want %q", got, token)
	}
}

func TestClearTokens(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	if err := StoreBotToken("xoxb-test"); err != nil {
		t.Fatalf("StoreBotToken() error: %v", err)
	}
	if err := StoreUserToken("xoxp-test"); err != nil {
		t.Fatalf("StoreUserToken() error: %v", err)
	}

	if err := ClearTokens(); err != nil {
		t.Fatalf("ClearTokens() error: %v", err)
	}

	bot, _ := GetBotToken()
	user, _ := GetUserToken()
	if bot != "" || user != "" {
		t.Errorf("after ClearTokens(): bot=%q user=%q, want both empty", bot, user)
	}
}

func TestGetBotToken_NoCredentials(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	_, err := GetBotToken()
	if err == nil {
		t.Fatal("GetBotToken() with no credentials should return an error")
	}
}

func TestGetUserToken_NoCredentials(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	_, err := GetUserToken()
	if err == nil {
		t.Fatal("GetUserToken() with no credentials should return an error")
	}
}

func TestKeyringFallbackToFile(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	// Capture stdout to suppress warning output during test.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	token := "xoxb-file-fallback"
	if err := StoreBotToken(token); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreBotToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	// Drain the pipe so it doesn't block.
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify the auth file was created.
	if _, err := os.Stat(config.AuthFile()); err != nil {
		t.Fatalf("auth file not created: %v", err)
	}

	got, err := GetBotToken()
	if err != nil {
		t.Fatalf("GetBotToken() from file fallback error: %v", err)
	}
	if got != token {
		t.Errorf("GetBotToken() = %q, want %q", got, token)
	}
}

func TestKeyringFallbackWarning(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := StoreBotToken("xoxb-test"); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreBotToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "plain text") {
		t.Errorf("expected warning about plain text storage, got: %q", output)
	}
}

func TestStoreAndGetTeamInfo(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	if err := StoreTeamInfo("T12345", "My Workspace"); err != nil {
		t.Fatalf("StoreTeamInfo() error: %v", err)
	}

	teamID, teamName := GetTeamInfo()
	if teamID != "T12345" {
		t.Errorf("GetTeamInfo() teamID = %q, want %q", teamID, "T12345")
	}
	if teamName != "My Workspace" {
		t.Errorf("GetTeamInfo() teamName = %q, want %q", teamName, "My Workspace")
	}
}

func TestStoreAndGetUserInfo(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	if err := StoreUserInfo("U12345", "isaac"); err != nil {
		t.Fatalf("StoreUserInfo() error: %v", err)
	}

	userID, userName := GetUserInfo()
	if userID != "U12345" {
		t.Errorf("GetUserInfo() userID = %q, want %q", userID, "U12345")
	}
	if userName != "isaac" {
		t.Errorf("GetUserInfo() userName = %q, want %q", userName, "isaac")
	}
}

func TestClearTokensFileFallback(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	// Capture stdout to suppress warning.
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	if err := StoreBotToken("xoxb-test"); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreBotToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	if err := ClearTokens(); err != nil {
		t.Fatalf("ClearTokens() error: %v", err)
	}

	bot, _ := GetBotToken()
	if bot != "" {
		t.Errorf("GetBotToken() after ClearTokens() = %q, want empty", bot)
	}
}
