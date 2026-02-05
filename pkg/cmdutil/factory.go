package cmdutil

import (
	"os"
	"sync"

	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
)

// Factory provides lazy-initialized dependencies to commands.
type Factory struct {
	IOStreams *iostreams.IOStreams

	configOnce sync.Once
	config     *config.Config
	configErr  error

	botClientOnce sync.Once
	botClient     *api.Client
	botClientErr  error

	userClientOnce sync.Once
	userClient     *api.Client
	userClientErr  error
}

// NewFactory creates a new Factory with the given IOStreams.
func NewFactory(ios *iostreams.IOStreams) *Factory {
	return &Factory{
		IOStreams: ios,
	}
}

// Config returns the loaded configuration (cached after first call).
func (f *Factory) Config() (*config.Config, error) {
	f.configOnce.Do(func() {
		f.config, f.configErr = config.Load()
	})
	return f.config, f.configErr
}

// BotClient returns a Slack API client authenticated with the bot token (cached).
func (f *Factory) BotClient() (*api.Client, error) {
	f.botClientOnce.Do(func() {
		token, err := auth.GetBotToken()
		if err != nil {
			f.botClientErr = err
			return
		}
		if token == "" {
			f.botClientErr = &TokenTypeError{Need: "bot"}
			return
		}
		f.botClient = api.NewClient(token)
	})
	return f.botClient, f.botClientErr
}

// UserClient returns a Slack API client authenticated with the user token (cached).
func (f *Factory) UserClient() (*api.Client, error) {
	f.userClientOnce.Do(func() {
		token, err := auth.GetUserToken()
		if err != nil {
			f.userClientErr = err
			return
		}
		if token == "" {
			f.userClientErr = &TokenTypeError{Need: "user"}
			return
		}
		f.userClient = api.NewClient(token)
	})
	return f.userClient, f.userClientErr
}

// DefaultClient returns the bot client if available, otherwise the user client.
func (f *Factory) DefaultClient() (*api.Client, error) {
	client, err := f.BotClient()
	if err == nil {
		return client, nil
	}
	return f.UserClient()
}

// IsAgentMode returns true when SLACKBUZZ_AGENT=1 is set.
// Agent mode prefers bot tokens, disables interactive prompts, and
// outputs structured JSON errors to stderr.
func (f *Factory) IsAgentMode() bool {
	return os.Getenv("SLACKBUZZ_AGENT") == "1"
}
