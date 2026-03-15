package text

import (
	"fmt"
	"regexp"
	"strings"
)

// Slack mrkdwn reference:
//   *bold*    _italic_    ~strikethrough~    `code`    ```code block```
//   > blockquote    • or - for bullets (- works natively)
//   <url|label> for links    :emoji: shortcodes
//   No headers (#), no **bold**, no [links](url), no ![images](url)

var (
	// Markdown bold: **text** or __text__ → *text*
	mdBoldDoubleAsterisk = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdBoldDoubleUnderscore = regexp.MustCompile(`__(.+?)__`)

	// Markdown italic: *text* is already valid mrkdwn, but _text_ also works.
	// We only need to handle cases where markdown uses single * for italic
	// and the user also has **bold** — the bold conversion handles that.

	// Markdown headers: # Header, ## Header, ### Header
	// Convert to *bold text* on its own line
	mdHeader = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

	// Markdown links: [text](url) → <url|text>
	mdLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	// Markdown images: ![alt](url) → <url|alt> (best effort)
	mdImage = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

	// Markdown horizontal rules: --- or *** or ___ on their own line
	// Convert to ───── divider (Slack has no native HR in mrkdwn text)
	mdHorizontalRule = regexp.MustCompile(`(?m)^[\s]*([-*_]){3,}\s*$`)

	// Markdown ordered list: 1. item → 1. item (already works, but detect for hints)
	mdOrderedList = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)

	// Detect markdown-style bold that won't render in Slack
	detectDoubleBold = regexp.MustCompile(`\*\*[^*]+\*\*`)

	// Detect markdown headers
	detectHeaders = regexp.MustCompile(`(?m)^#{1,6}\s+`)

	// Detect markdown links
	detectLinks = regexp.MustCompile(`\[[^\]]+\]\([^)]+\)`)

	// Detect markdown images
	detectImages = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
)

// FormatHint describes a formatting issue detected in the message text.
type FormatHint struct {
	Issue      string // What was detected
	Suggestion string // How to fix it for Slack
	Example    string // Before → After example
}

// ConvertMarkdownToMrkdwn converts common Markdown syntax to Slack mrkdwn.
// It handles bold, headers, links, images, and horizontal rules.
// Already-valid mrkdwn (like *bold*, _italic_, ~strike~, `code`) passes through unchanged.
func ConvertMarkdownToMrkdwn(text string) string {
	// Order matters: images before links (images contain link syntax)

	// Images: ![alt](url) → <url|alt> or just <url>
	text = mdImage.ReplaceAllStringFunc(text, func(match string) string {
		parts := mdImage.FindStringSubmatch(match)
		alt, url := parts[1], parts[2]
		if alt != "" {
			return fmt.Sprintf("<%s|%s>", url, alt)
		}
		return fmt.Sprintf("<%s>", url)
	})

	// Links: [text](url) → <url|text>
	text = mdLink.ReplaceAllStringFunc(text, func(match string) string {
		parts := mdLink.FindStringSubmatch(match)
		label, url := parts[1], parts[2]
		return fmt.Sprintf("<%s|%s>", url, label)
	})

	// Headers first: # Text → *Text* (before bold, so nested **bold** in headers
	// gets cleaned up by the bold pass)
	text = mdHeader.ReplaceAllStringFunc(text, func(match string) string {
		parts := mdHeader.FindStringSubmatch(match)
		// Strip any ** markers inside the header text before wrapping in *
		inner := mdBoldDoubleAsterisk.ReplaceAllString(parts[2], "$1")
		inner = mdBoldDoubleUnderscore.ReplaceAllString(inner, "$1")
		return "*" + inner + "*"
	})

	// Bold: **text** → *text*
	text = mdBoldDoubleAsterisk.ReplaceAllString(text, "*$1*")
	text = mdBoldDoubleUnderscore.ReplaceAllString(text, "*$1*")

	// Horizontal rules: --- → ─────
	text = mdHorizontalRule.ReplaceAllString(text, "─────")

	return text
}

// DetectFormatHints inspects message text and returns hints about Markdown
// syntax that won't render correctly in Slack. Call this BEFORE conversion
// to give the user actionable feedback.
func DetectFormatHints(text string) []FormatHint {
	var hints []FormatHint

	if detectDoubleBold.MatchString(text) {
		hints = append(hints, FormatHint{
			Issue:      "Markdown bold (**text**) won't render in Slack",
			Suggestion: "Use *text* for bold in Slack mrkdwn",
			Example:    "**bold** → *bold*",
		})
	}

	if detectHeaders.MatchString(text) {
		hints = append(hints, FormatHint{
			Issue:      "Markdown headers (# Header) aren't supported in Slack",
			Suggestion: "Use *bold text* on its own line for emphasis",
			Example:    "# Section → *Section*",
		})
	}

	if detectLinks.MatchString(text) {
		hints = append(hints, FormatHint{
			Issue:      "Markdown links [text](url) won't render in Slack",
			Suggestion: "Use <url|text> for Slack links",
			Example:    "[Click here](https://...) → <https://...|Click here>",
		})
	}

	if detectImages.MatchString(text) {
		hints = append(hints, FormatHint{
			Issue:      "Markdown images ![alt](url) aren't supported in Slack",
			Suggestion: "Use <url|alt> — Slack will unfurl image URLs automatically",
			Example:    "![logo](https://...) → <https://...|logo>",
		})
	}

	return hints
}

// SlackMrkdwnReference returns a quick-reference string for Slack mrkdwn syntax.
func SlackMrkdwnReference() string {
	return strings.TrimSpace(`
Slack mrkdwn formatting reference:
  *bold*           Bold text
  _italic_         Italic text
  ~strikethrough~  Strikethrough
  ` + "`code`" + `          Inline code
  ` + "```code```" + `      Code block
  > quote          Blockquote
  • item           Bullet list (or - item)
  1. item          Numbered list
  <url|label>      Hyperlink
  :emoji:          Emoji shortcode
  @user            Mention (auto-resolved by slackbuzz)
`)
}
