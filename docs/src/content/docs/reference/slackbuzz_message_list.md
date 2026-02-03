---
title: "slackbuzz message list"
description: "Auto-generated reference for slackbuzz message list"
---

## slackbuzz message list

List messages in a channel

### Synopsis

Read message history from a Slack channel or thread.

The channel argument accepts #channel-name or a channel ID.

```
slackbuzz message list <channel> [flags]
```

### Examples

```
  # List recent messages
  slackbuzz message list #general

  # List with limit
  slackbuzz message list #general --limit 20

  # List since a date
  slackbuzz message list #general --since 2026-01-01

  # List thread replies
  slackbuzz message list #general --thread 1234567890.123456

  # Output as JSON
  slackbuzz message list #general --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Number of messages to fetch (default 10)
      --since string      Only show messages after this date (YYYY-MM-DD)
      --template string   Format JSON output using a Go template
      --thread string     Thread timestamp to read replies from
```

### SEE ALSO

* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages

