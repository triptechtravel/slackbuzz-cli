---
title: "slackbuzz message edit"
description: "Auto-generated reference for slackbuzz message edit"
---

## slackbuzz message edit

Edit a message

### Synopsis

Update the text of an existing Slack message.

The channel argument accepts #channel-name or a channel ID.
If new-text is omitted, reads from stdin (for piping).

```
slackbuzz message edit <channel> <timestamp> [new-text] [flags]
```

### Examples

```
  # Edit a message
  slackbuzz message edit #general 1706000000.000000 "updated text"

  # Pipe new text
  echo "corrected text" | slackbuzz message edit #general 1706000000.000000
```

### Options

```
  -h, --help              help for edit
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages

