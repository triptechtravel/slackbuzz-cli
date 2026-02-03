package text

import (
	"regexp"
	"strings"
)

var (
	// <@U123|name> or <@U123> → @name or @U123
	userMentionRe = regexp.MustCompile(`<@([UW][A-Z0-9]+)(?:\|([^>]+))?>`)
	// <#C123|channel> or <#C123> → #channel or #C123
	channelMentionRe = regexp.MustCompile(`<#([CG][A-Z0-9]+)(?:\|([^>]+))?>`)
	// <url|label> → label (url) or just url
	linkRe = regexp.MustCompile(`<(https?://[^|>]+)(?:\|([^>]+))?>`)
	// <!subteam^ID|@handle> → @handle
	groupMentionRe = regexp.MustCompile(`<!subteam\^[A-Z0-9]+\|(@[^>]+)>`)
	// <!here>, <!channel>, <!everyone>
	broadcastRe = regexp.MustCompile(`<!([a-z]+)(?:\|([^>]+))?>`)
)

// FormatSlackText converts Slack mrkdwn to a terminal-friendly string.
// It resolves user mentions, channel references, URLs, and emoji shortcodes.
func FormatSlackText(s string) string {
	// User mentions: <@U123|name> → @name
	s = userMentionRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := userMentionRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return "@" + parts[2]
		}
		return "@" + parts[1]
	})

	// Channel mentions: <#C123|channel> → #channel
	s = channelMentionRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := channelMentionRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return "#" + parts[2]
		}
		return "#" + parts[1]
	})

	// Group mentions: <!subteam^ID|@handle> → @handle
	s = groupMentionRe.ReplaceAllString(s, "$1")

	// Broadcast mentions: <!here> → @here, <!channel> → @channel
	s = broadcastRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := broadcastRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return "@" + parts[2]
		}
		return "@" + parts[1]
	})

	// Links: <url|label> → label, <url> → url
	s = linkRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return parts[2]
		}
		return parts[1]
	})

	// Emoji shortcodes → Unicode (uses custom workspace emoji if loaded)
	if customEmojiMap != nil {
		s = ReplaceEmojiWithCustom(s)
	} else {
		s = ReplaceEmoji(s)
	}

	// Slack encodes &amp; &lt; &gt;
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")

	return s
}
