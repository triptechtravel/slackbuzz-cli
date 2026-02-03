---
title: "slackbuzz message send"
description: "Auto-generated reference for slackbuzz message send"
---

## slackbuzz message send

Send a message to a channel

### Synopsis

Post a message to a Slack channel, DM, or thread.

The channel argument accepts #channel-name or a channel ID.
If text is omitted, reads from stdin (for piping).

```
slackbuzz message send <channel> [text] [flags]
```

### Examples

```
  # Send a message
  slackbuzz message send #general "Hello, world!"

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

