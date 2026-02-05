package auth

import (
	"fmt"

	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	serviceName    = "slack-cli"
	botTokenKey    = "bot_token"
	userTokenKey   = "user_token"
	teamIDKey      = "team_id"
	teamNameKey    = "team_name"
	userIDKey      = "user_id"
	userNameKey    = "user_name"
	botUserIDKey   = "bot_user_id"
	botUserNameKey = "bot_user_name"
)

// StoreBotToken saves a bot token (xoxb-) to the OS keyring, falling back to plaintext.
func StoreBotToken(token string) error {
	return storeKeyringValue(botTokenKey, token, func(ac *config.AuthConfig) {
		ac.BotToken = token
	})
}

// StoreUserToken saves a user token (xoxp-) to the OS keyring, falling back to plaintext.
func StoreUserToken(token string) error {
	return storeKeyringValue(userTokenKey, token, func(ac *config.AuthConfig) {
		ac.UserToken = token
	})
}

// StoreTeamInfo saves the team ID and name.
func StoreTeamInfo(teamID, teamName string) error {
	_ = keyring.Set(serviceName, teamIDKey, teamID)
	_ = keyring.Set(serviceName, teamNameKey, teamName)

	// Also persist to file as fallback
	ac, _ := config.LoadAuth()
	ac.TeamID = teamID
	ac.TeamName = teamName
	return ac.Save()
}

// ResolveUserID returns the human user's Slack ID, fetching it via auth.test
// on the user token if not already stored. Only uses the user token (not bot)
// to ensure the returned ID is the human's, not the bot's.
func ResolveUserID() (userID, userName string, err error) {
	userID, userName = GetUserInfo()
	if userID != "" {
		return userID, userName, nil
	}

	// Only use user token — bot token would return the bot's identity
	token, tokenErr := GetUserToken()
	if tokenErr != nil || token == "" {
		return "", "", fmt.Errorf("no user token available to resolve user ID (bot token returns bot identity, not yours)")
	}

	info, err := ValidateToken(token)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve user ID: %w", err)
	}

	// Cache for next time
	_ = StoreUserInfo(info.UserID, info.User)
	return info.UserID, info.User, nil
}

// GetBotToken retrieves the stored bot token.
func GetBotToken() (string, error) {
	return getKeyringValue(botTokenKey, func(ac *config.AuthConfig) string {
		return ac.BotToken
	})
}

// GetUserToken retrieves the stored user token.
func GetUserToken() (string, error) {
	return getKeyringValue(userTokenKey, func(ac *config.AuthConfig) string {
		return ac.UserToken
	})
}

// StoreUserInfo saves the human user's Slack ID and username (from user token auth.test).
func StoreUserInfo(userID, userName string) error {
	_ = keyring.Set(serviceName, userIDKey, userID)
	_ = keyring.Set(serviceName, userNameKey, userName)

	// Also persist to file as fallback
	ac, _ := config.LoadAuth()
	ac.UserID = userID
	ac.UserName = userName
	return ac.Save()
}

// StoreBotUserInfo saves the bot's Slack user ID and username (from bot token auth.test).
func StoreBotUserInfo(userID, userName string) error {
	_ = keyring.Set(serviceName, botUserIDKey, userID)
	_ = keyring.Set(serviceName, botUserNameKey, userName)

	// Also persist to file as fallback
	ac, _ := config.LoadAuth()
	ac.BotUserID = userID
	ac.BotUserName = userName
	return ac.Save()
}

// GetUserInfo retrieves the stored human user ID and username.
func GetUserInfo() (userID, userName string) {
	userID, _ = keyring.Get(serviceName, userIDKey)
	userName, _ = keyring.Get(serviceName, userNameKey)
	if userID != "" {
		return
	}

	ac, err := config.LoadAuth()
	if err != nil {
		return "", ""
	}
	return ac.UserID, ac.UserName
}

// GetBotUserInfo retrieves the stored bot user ID and username.
func GetBotUserInfo() (userID, userName string) {
	userID, _ = keyring.Get(serviceName, botUserIDKey)
	userName, _ = keyring.Get(serviceName, botUserNameKey)
	if userID != "" {
		return
	}

	ac, err := config.LoadAuth()
	if err != nil {
		return "", ""
	}
	return ac.BotUserID, ac.BotUserName
}

// GetTeamInfo retrieves the stored team ID and name.
func GetTeamInfo() (teamID, teamName string) {
	teamID, _ = keyring.Get(serviceName, teamIDKey)
	teamName, _ = keyring.Get(serviceName, teamNameKey)
	if teamID != "" {
		return
	}

	ac, err := config.LoadAuth()
	if err != nil {
		return "", ""
	}
	return ac.TeamID, ac.TeamName
}

// ClearTokens removes all stored credentials from keyring and file.
func ClearTokens() error {
	_ = keyring.Delete(serviceName, botTokenKey)
	_ = keyring.Delete(serviceName, userTokenKey)
	_ = keyring.Delete(serviceName, teamIDKey)
	_ = keyring.Delete(serviceName, teamNameKey)
	_ = keyring.Delete(serviceName, userIDKey)
	_ = keyring.Delete(serviceName, userNameKey)
	_ = keyring.Delete(serviceName, botUserIDKey)
	_ = keyring.Delete(serviceName, botUserNameKey)

	ac := &config.AuthConfig{}
	_ = ac.Clear()
	return nil
}

func storeKeyringValue(key, value string, fallbackSetter func(*config.AuthConfig)) error {
	err := keyring.Set(serviceName, key, value)
	if err == nil {
		return nil
	}

	// Fallback to file-based storage
	ac, loadErr := config.LoadAuth()
	if loadErr != nil {
		ac = &config.AuthConfig{}
	}
	fallbackSetter(ac)
	if saveErr := ac.Save(); saveErr != nil {
		return fmt.Errorf("failed to store token: keyring error: %w, file error: %v", err, saveErr)
	}
	fmt.Println("Warning: Could not use OS keyring. Token stored in plain text at", config.AuthFile())
	return nil
}

func getKeyringValue(key string, fallbackGetter func(*config.AuthConfig) string) (string, error) {
	token, err := keyring.Get(serviceName, key)
	if err == nil && token != "" {
		return token, nil
	}

	// Fallback to file-based storage
	ac, err := config.LoadAuth()
	if err != nil {
		return "", fmt.Errorf("no stored credentials found: %w", err)
	}
	value := fallbackGetter(ac)
	if value == "" {
		return "", fmt.Errorf("not authenticated. Run 'slackbuzz auth login' to authenticate")
	}
	return value, nil
}
