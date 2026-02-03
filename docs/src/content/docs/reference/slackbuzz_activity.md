---
title: "slackbuzz activity"
description: "Auto-generated reference for slackbuzz activity"
---

## slackbuzz activity

Show what needs your attention — mentions, DMs, threads

### Synopsis

Show messages that need your attention: mentions, DMs, and threads.

By default shows messages where you were mentioned. Use flags to filter.
Detects ClickUp task IDs and GitHub PR/issue URLs and shows actionable hints.

Requires a user token (xoxp-) for the search API.

```
slackbuzz activity [flags]
```

### Examples

```
  # Show mentions (default)
  slackbuzz activity

  # Show DMs
  slackbuzz activity --dms

  # Show threads you're in
  slackbuzz activity --threads

  # Everything from the last day
  slackbuzz activity --all --since 1d

  # Filter by channel or user
  slackbuzz activity --channel #engineering --from @alice

  # Output as JSON
  slackbuzz activity --json
```

### Options

```
      --all               Show mentions, DMs, and threads combined
      --channel string    Filter by channel (e.g. #engineering)
      --dms               Show direct messages
      --from string       Filter by sender (e.g. @alice)
  -h, --help              help for activity
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of results (default 20)
      --since string      Only show messages after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)
      --template string   Format JSON output using a Go template
      --threads           Show threads you're mentioned in
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

