---
title: "slackbuzz message search"
description: "Auto-generated reference for slackbuzz message search"
---

## slackbuzz message search

Search messages (requires user token)

### Synopsis

Search Slack messages using the search.messages API.

This command requires a user token (xoxp-) because the search API
does not support bot tokens.

```
slackbuzz message search <query> [flags]
```

### Examples

```
  # Search for messages
  slackbuzz message search "deploy production"

  # Search in a specific channel
  slackbuzz message search "error" --channel #general

  # Search from a user since a date
  slackbuzz message search "fix" --from @alice --since 2026-01-01
```

### Options

```
      --channel string    Filter by channel
      --from string       Filter by user
  -h, --help              help for search
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of results (default 20)
      --since string      Only show messages after this date (YYYY-MM-DD)
      --template string   Format JSON output using a Go template
      --until string      Only show messages before this date (YYYY-MM-DD)
```

### SEE ALSO

* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages

