---
title: "slackbuzz threads"
description: "Auto-generated reference for slackbuzz threads"
---

## slackbuzz threads

Show threads you're participating in

### Synopsis

List threads where you've posted, mirroring Slack's Threads sidebar.

Uses the search API to find threads where you're active.
Requires a user token (xoxp-).

```
slackbuzz threads [flags]
```

### Examples

```
  # Recent threads
  slackbuzz threads

  # Threads from the last day
  slackbuzz threads --since 1d

  # Output as JSON
  slackbuzz threads --json
```

### Options

```
  -h, --help              help for threads
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of results (default 20)
      --since string      Only show threads after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

