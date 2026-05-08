// Block builders + a few helpers for things Slack's flat OpenAPI doesn't
// cover (block kit composition, message references, reaction targets).
//
// Pure data — every type here marshals straight to the JSON shape Slack
// expects on the wire. Pass a `[]Block` to chat.postMessage / chat.update
// via SerializeBlocks(), or use BlocksJSON() inline if you only need the
// string.
package slackapi

import (
	"encoding/json"
	"fmt"
)

// ─── Text objects ─────────────────────────────────────────────────────────

// TextObject is a Slack composition object — used inside blocks.
//
// Type is "mrkdwn" (Slack's markdown-ish format) or "plain_text".
type TextObject struct {
	Type     string `json:"type"` // "mrkdwn" | "plain_text"
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`    // plain_text only
	Verbatim bool   `json:"verbatim,omitempty"` // mrkdwn only
}

// NewTextObject constructs a TextObject. Mirrors slack-go's
// NewTextBlockObject(typeStr, text, emoji, verbatim).
func NewTextObject(typeStr, text string, emoji, verbatim bool) *TextObject {
	return &TextObject{Type: typeStr, Text: text, Emoji: emoji, Verbatim: verbatim}
}

// NewMrkdwnText is shorthand for NewTextObject("mrkdwn", text, false, false).
func NewMrkdwnText(text string) *TextObject {
	return &TextObject{Type: "mrkdwn", Text: text}
}

// NewPlainText is shorthand for NewTextObject("plain_text", text, true, false).
func NewPlainText(text string) *TextObject {
	return &TextObject{Type: "plain_text", Text: text, Emoji: true}
}

// ─── Blocks ────────────────────────────────────────────────────────────────

// Block is the interface every concrete block type implements. Slack
// distinguishes blocks by their `type` field on the wire; each concrete
// type carries that field as a json:"type" tag.
type Block interface {
	blockType() string
}

// SectionBlock is a generic section — text, optional fields, optional accessory.
type SectionBlock struct {
	Type      string                 `json:"type"` // always "section"
	Text      *TextObject            `json:"text,omitempty"`
	Fields    []*TextObject          `json:"fields,omitempty"`
	Accessory map[string]interface{} `json:"accessory,omitempty"`
	BlockID   string                 `json:"block_id,omitempty"`
}

func (s *SectionBlock) blockType() string { return "section" }

// NewSectionBlock constructs a section block. Matches slack-go's
// NewSectionBlock(text, fields, accessory) signature.
func NewSectionBlock(text *TextObject, fields []*TextObject, accessory map[string]interface{}) *SectionBlock {
	return &SectionBlock{Type: "section", Text: text, Fields: fields, Accessory: accessory}
}

// HeaderBlock displays a large heading.
type HeaderBlock struct {
	Type    string      `json:"type"` // always "header"
	Text    *TextObject `json:"text"` // must be plain_text
	BlockID string      `json:"block_id,omitempty"`
}

func (h *HeaderBlock) blockType() string { return "header" }

// NewHeaderBlock constructs a header block from a plain_text TextObject.
func NewHeaderBlock(text *TextObject) *HeaderBlock {
	return &HeaderBlock{Type: "header", Text: text}
}

// DividerBlock is a visual separator.
type DividerBlock struct {
	Type    string `json:"type"` // always "divider"
	BlockID string `json:"block_id,omitempty"`
}

func (d *DividerBlock) blockType() string { return "divider" }

// NewDividerBlock constructs a divider block.
func NewDividerBlock() *DividerBlock {
	return &DividerBlock{Type: "divider"}
}

// ContextBlock holds a small line of context elements (text + images).
type ContextBlock struct {
	Type     string        `json:"type"` // always "context"
	Elements []interface{} `json:"elements"`
	BlockID  string        `json:"block_id,omitempty"`
}

func (c *ContextBlock) blockType() string { return "context" }

// NewContextBlock constructs a context block from heterogeneous elements
// (TextObject pointers, image objects, etc.).
func NewContextBlock(elements ...interface{}) *ContextBlock {
	return &ContextBlock{Type: "context", Elements: elements}
}

// ─── Block list serialization ─────────────────────────────────────────────

// BlocksJSON serializes a slice of blocks to the JSON-string form Slack's
// chat.postMessage / chat.update expect on the `blocks` form parameter.
//
// Returns "" for an empty input so callers can pass it directly without
// special-casing.
func BlocksJSON(blocks []Block) string {
	if len(blocks) == 0 {
		return ""
	}
	out, err := json.Marshal(blocks)
	if err != nil {
		// Block construction is purely in-memory — a marshal failure here
		// is a programmer error, not a runtime condition. Surface it
		// loudly via fmt.Sprint rather than silently dropping.
		return fmt.Sprintf("[]/* marshal failed: %v */", err)
	}
	return string(out)
}

// ─── Reactions / item references ──────────────────────────────────────────

// ItemRef identifies a message-target for the reactions.* and pins.* family
// of methods (channel + timestamp).
type ItemRef struct {
	Channel   string
	Timestamp string
}

// NewRefToMessage mirrors slack-go's NewRefToMessage(channelID, ts).
func NewRefToMessage(channel, timestamp string) ItemRef {
	return ItemRef{Channel: channel, Timestamp: timestamp}
}
