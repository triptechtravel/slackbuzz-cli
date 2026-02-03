---
title: "slackbuzz channel list"
description: "Auto-generated reference for slackbuzz channel list"
---

## slackbuzz channel list

List channels

### Synopsis

List Slack channels in the workspace.

```
slackbuzz channel list [flags]
```

### Examples

```
  # List public channels
  slackbuzz channel list

  # List private channels
  slackbuzz channel list --type private

  # List DMs
  slackbuzz channel list --type im

  # Output as JSON
  slackbuzz channel list --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of channels to list (default 100)
      --template string   Format JSON output using a Go template
      --type string       Channel type: public, private, im, mpim (default "public")
```

### SEE ALSO

* [slackbuzz channel](/slackbuzz-cli/reference/slackbuzz_channel/)	 - Manage Slack channels

