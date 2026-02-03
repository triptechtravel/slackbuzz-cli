---
title: "slackbuzz dm list"
description: "Auto-generated reference for slackbuzz dm list"
---

## slackbuzz dm list

List DM conversations with recent activity

### Synopsis

Show your DM conversations grouped by contact.

Uses the search API to find recent DMs.
Requires a user token (xoxp-).

```
slackbuzz dm list [flags]
```

### Examples

```
  # List recent DMs
  slackbuzz dm list

  # DMs from the last day
  slackbuzz dm list --since 1d

  # Output as JSON
  slackbuzz dm list --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of conversations (default 20)
      --since string      Only show DMs after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz dm](/slackbuzz-cli/reference/slackbuzz_dm/)	 - Direct message management

