package text

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	customEmojiOnce sync.Once
	customEmojiMap  map[string]string // shortcode → URL or "alias:name"
)

// LoadCustomEmoji fetches the workspace's custom emoji via the emoji.list API
// and merges aliases into the emoji lookup. Image-only custom emoji are stored
// as-is (the shortcode is preserved since we can't render images in terminal).
// This is called once per session; subsequent calls are no-ops.
func LoadCustomEmoji(token string) {
	customEmojiOnce.Do(func() {
		customEmojiMap = fetchEmojiList(token)
	})
}

// LookupEmoji resolves a shortcode (without colons) to a Unicode emoji.
// It checks custom workspace emoji first (resolving aliases), then falls back
// to the built-in gemoji map.
func LookupEmoji(code string) (string, bool) {
	// Check custom emoji first
	if customEmojiMap != nil {
		if val, ok := customEmojiMap[code]; ok {
			if strings.HasPrefix(val, "alias:") {
				// Resolve alias — look up the target in built-in map
				target := strings.TrimPrefix(val, "alias:")
				if emoji, ok := emojiMap[target]; ok {
					return emoji, true
				}
				// Alias to another custom emoji (one level)
				if val2, ok := customEmojiMap[target]; ok && !strings.HasPrefix(val2, "alias:") {
					// Image URL — not renderable
					return "", false
				}
				return "", false
			}
			// Image URL — not renderable in terminal
			return "", false
		}
	}

	// Fall back to built-in gemoji
	if emoji, ok := emojiMap[code]; ok {
		return emoji, true
	}
	return "", false
}

func fetchEmojiList(token string) map[string]string {
	v := url.Values{}
	v.Set("token", token)

	resp, err := http.PostForm("https://slack.com/api/emoji.list", v)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		OK    bool              `json:"ok"`
		Error string            `json:"error,omitempty"`
		Emoji map[string]string `json:"emoji"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.OK {
		return nil
	}

	return result.Emoji
}

// ReplaceEmojiWithCustom converts Slack :emoji: shortcodes to Unicode emoji,
// using both the workspace custom emoji and the built-in gemoji map.
func ReplaceEmojiWithCustom(s string) string {
	return emojiRe.ReplaceAllStringFunc(s, func(match string) string {
		code := strings.Trim(match, ":")
		if emoji, ok := LookupEmoji(code); ok {
			return emoji
		}
		return match
	})
}

// EmojiToUnicode converts a single emoji shortcode (with or without colons) to Unicode.
// Returns the shortcode wrapped in colons if not found.
func EmojiToUnicode(s string) string {
	code := strings.Trim(s, ":")
	if emoji, ok := LookupEmoji(code); ok {
		return emoji
	}
	return fmt.Sprintf(":%s:", code)
}
