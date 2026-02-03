---
title: "slackbuzz message delete"
description: "Auto-generated reference for slackbuzz message delete"
---

## slackbuzz message delete

Delete a message

### Synopsis

Delete a Slack message by channel and timestamp.

Prompts for confirmation in interactive mode.
In non-interactive (piped) mode, --confirm is required.

```
slackbuzz message delete <channel> <timestamp> [flags]
```

### Examples

```
  # Delete with confirmation prompt
  slackbuzz message delete #general 1706000000.000000

  # Delete without prompt (scripting)
  slackbuzz message delete #general 1706000000.000000 --confirm
```

### Options

```
      --as-bot            Delete as the bot instead of your user account
      --confirm           Skip confirmation prompt
  -h, --help              help for delete
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages

