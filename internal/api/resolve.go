package api

import (
	"fmt"
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

// Resolver maps human-friendly channel names (#general) and user names (@alice)
// to their Slack IDs. Results are cached for the lifetime of the process.
type Resolver struct {
	client   *slack.Client
	mu       sync.Mutex
	channels map[string]string // name → ID
	users    map[string]string // name → ID
	loaded   bool
}

// NewResolver creates a new Resolver backed by the given Slack client.
func NewResolver(client *slack.Client) *Resolver {
	return &Resolver{
		client:   client,
		channels: make(map[string]string),
		users:    make(map[string]string),
	}
}

// ResolveChannel takes a channel name (with or without #) or a channel ID
// and returns the channel ID.
func (r *Resolver) ResolveChannel(nameOrID string) (string, error) {
	// Strip leading # if present
	nameOrID = strings.TrimPrefix(nameOrID, "#")

	// If it looks like a channel ID (starts with C, G, or D), return as-is
	if isChannelID(nameOrID) {
		return nameOrID, nil
	}

	r.mu.Lock()
	if id, ok := r.channels[nameOrID]; ok {
		r.mu.Unlock()
		return id, nil
	}
	r.mu.Unlock()

	// Load channels if not yet cached
	if err := r.loadChannels(); err != nil {
		return "", fmt.Errorf("failed to resolve channel %q: %w", nameOrID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.channels[nameOrID]; ok {
		return id, nil
	}

	return "", fmt.Errorf("channel %q not found", nameOrID)
}

// ResolveUser takes a username (with or without @) or a user ID and returns the user ID.
func (r *Resolver) ResolveUser(nameOrID string) (string, error) {
	// Strip leading @ if present
	nameOrID = strings.TrimPrefix(nameOrID, "@")

	// If it looks like a user ID (starts with U or W), return as-is
	if isUserID(nameOrID) {
		return nameOrID, nil
	}

	r.mu.Lock()
	if id, ok := r.users[nameOrID]; ok {
		r.mu.Unlock()
		return id, nil
	}
	r.mu.Unlock()

	// Load users if not yet cached
	if err := r.loadUsers(); err != nil {
		return "", fmt.Errorf("failed to resolve user %q: %w", nameOrID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.users[nameOrID]; ok {
		return id, nil
	}

	return "", fmt.Errorf("user %q not found", nameOrID)
}

func (r *Resolver) loadChannels() error {
	r.mu.Lock()
	if r.loaded {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	params := &slack.GetConversationsParameters{
		Types:           []string{"public_channel", "private_channel", "im", "mpim"},
		Limit:           1000,
		ExcludeArchived: true,
	}

	for {
		channels, nextCursor, err := r.client.GetConversations(params)
		if err != nil {
			return err
		}

		r.mu.Lock()
		for _, ch := range channels {
			r.channels[ch.Name] = ch.ID
		}
		r.mu.Unlock()

		if nextCursor == "" {
			break
		}
		params.Cursor = nextCursor
	}

	r.mu.Lock()
	r.loaded = true
	r.mu.Unlock()

	return nil
}

// ResolveDM takes a username, @username, or user ID and returns a DM channel ID.
// It resolves the user and opens (or retrieves) a DM conversation via conversations.open.
func (r *Resolver) ResolveDM(nameOrID string) (string, error) {
	userID, err := r.ResolveUser(nameOrID)
	if err != nil {
		return "", err
	}

	ch, _, _, err := r.client.OpenConversation(&slack.OpenConversationParameters{
		Users: []string{userID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to open DM with %q: %w", nameOrID, err)
	}
	return ch.ID, nil
}

// LooksLikeUser returns true if the target looks like a user reference
// (starts with @, starts with U/W, or is not #-prefixed and not a channel ID).
func LooksLikeUser(target string) bool {
	if strings.HasPrefix(target, "@") {
		return true
	}
	if isUserID(target) {
		return true
	}
	if strings.HasPrefix(target, "#") || isChannelID(target) {
		return false
	}
	// Bare name — assume user if it doesn't look like a channel ID
	return true
}

func (r *Resolver) loadUsers() error {
	users, err := r.client.GetUsers()
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range users {
		if !u.Deleted {
			r.users[u.Name] = u.ID
			lower := strings.ToLower(u.Name)
			if lower != u.Name {
				r.users[lower] = u.ID
			}
			if u.Profile.DisplayName != "" {
				r.users[u.Profile.DisplayName] = u.ID
				lowerDisplay := strings.ToLower(u.Profile.DisplayName)
				if lowerDisplay != u.Profile.DisplayName {
					r.users[lowerDisplay] = u.ID
				}
			}
		}
	}
	return nil
}

func isChannelID(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[0] == 'C' || s[0] == 'G' || s[0] == 'D'
}

func isUserID(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[0] == 'U' || s[0] == 'W'
}
