---
title: "slackbuzz notify"
description: "Auto-generated reference for slackbuzz notify"
---

## slackbuzz notify

Send formatted notifications

### Synopsis

Send formatted notification messages to a Slack channel.

Supports release announcements, task status updates, and custom messages.

```
slackbuzz notify <channel> [flags]
```

### Examples

```
  # Release announcement
  slackbuzz notify #releases --release v1.0.0

  # Task status update
  slackbuzz notify #updates --task CU-abc123 --status "deployed"

  # Custom message
  slackbuzz notify #general --message "Maintenance window starting"
```

### Options

```
  -h, --help              help for notify
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --message string    Custom notification message
      --release string    Release version to announce
      --status string     Task status (used with --task)
      --task string       ClickUp task ID for status update
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

