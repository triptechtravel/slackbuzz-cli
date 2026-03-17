---
title: "slackbuzz send"
description: "Auto-generated reference for slackbuzz send"
---

## slackbuzz send

Send a message (shortcut for 'message send')

### Synopsis

Post a message to a Slack channel, DM, or thread.

The first argument accepts a #channel-name, channel ID, @username, or user ID.
To send a DM, use a username, @username, or user ID as the channel argument.
If text is omitted, reads from stdin (for piping).

Formatting: Standard Markdown (**bold**, # headers, [links](url)) is
automatically converted to Slack mrkdwn (*bold*, etc). Use --raw to
disable auto-conversion and send text as-is.

Use --blocks to send via Block Kit for richer formatting (sections with
mrkdwn rendering). Long messages are automatically split into multiple
blocks (Slack's 3000-char limit per block).

```
slackbuzz send <channel|user> [text] [flags]
```

### Examples

```
  # Send a message to a channel
  slackbuzz message send #general "Hello, world!"

  # Send a DM by @username
  slackbuzz message send @sarah "Quick question about the API"

  # Send with Slack mrkdwn formatting (auto-converted from Markdown)
  slackbuzz message send #releases "## 🚀 v5.4.0\n- **New feature**: offline images"

  # Send via Block Kit for richer rendering
  slackbuzz message send #releases --blocks "## Release Notes\n- *Feature*: offline images"

  # Send raw text without Markdown→mrkdwn conversion
  slackbuzz message send #dev --raw "**this stays as double asterisks**"

  # Send to a thread
  slackbuzz message send #general "Reply here" --thread-ts 1234567890.123456

  # Pipe content
  echo "Build passed" | slackbuzz message send #deploys

  # Pipe from a command
  git log --oneline -5 | slackbuzz message send #dev-logs
```

### Options

```
      --as-bot             Send as the bot instead of your user account
      --blocks             Send via Block Kit sections (richer mrkdwn rendering)
  -h, --help               help for send
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --raw                Disable Markdown→mrkdwn auto-conversion
      --template string    Format JSON output using a Go template
      --thread-ts string   Thread timestamp to reply to
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

