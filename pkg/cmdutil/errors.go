package cmdutil

import "errors"

// SilentError is an error that should not be printed (the command already handled output).
type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	return e.Err
}

// AuthError indicates an authentication failure.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "authentication required. Run 'slackbuzz auth login' to authenticate"
}

// TokenTypeError indicates the wrong token type was used.
type TokenTypeError struct {
	Need    string // "bot" or "user"
	Message string
}

func (e *TokenTypeError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	switch e.Need {
	case "user":
		return "this command requires a user token (xoxp-). Run 'slackbuzz auth login --user-token' to add one"
	case "bot":
		return "this command requires a bot token (xoxb-). Run 'slackbuzz auth login --bot-token' to add one"
	default:
		return "wrong token type for this command"
	}
}

// IsSilentError checks if the error is a SilentError.
func IsSilentError(err error) bool {
	var se *SilentError
	return errors.As(err, &se)
}

// IsAuthError checks if the error is an AuthError.
func IsAuthError(err error) bool {
	var ae *AuthError
	return errors.As(err, &ae)
}

// IsTokenTypeError checks if the error is a TokenTypeError.
func IsTokenTypeError(err error) bool {
	var te *TokenTypeError
	return errors.As(err, &te)
}
