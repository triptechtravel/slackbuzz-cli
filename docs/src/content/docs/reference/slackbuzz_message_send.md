---
title: "slackbuzz message send"
description: "Auto-generated reference for slackbuzz message send"
---

## slackbuzz message send

Send a message to a channel or DM

### Synopsis

Post a message to a Slack channel, DM, or thread.

The first argument accepts a #channel-name, channel ID, @username, or user ID.
To send a DM, use a username, @username, or user ID as the channel argument.
If text is omitted, reads from stdin (for piping).

```
slackbuzz message send <channel|user> [text] [flags]
```

### Examples

```
  # Send a message to a channel
  slackbuzz message send #general "Hello, world!"

  # Send a DM by @username
  slackbuzz message send @sarah "Quick question about the API"

  # Send a DM by username (no @ needed)
  slackbuzz message send herman "Hey, got a minute?"

  # Send a DM by user ID
  slackbuzz message send U02P3QC5H24 "Direct message by ID"

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
  -h, --help               help for send
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --template string    Format JSON output using a Go template
      --thread-ts string   Thread timestamp to reply to
```

### SEE ALSO

* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages

