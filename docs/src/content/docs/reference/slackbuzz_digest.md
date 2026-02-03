---
title: "slackbuzz digest"
description: "Auto-generated reference for slackbuzz digest"
---

## slackbuzz digest

Cross-tool morning briefing (Slack + ClickUp + GitHub)

### Synopsis

Show a combined briefing of what needs your attention across Slack,
ClickUp, and GitHub.

Slack data comes from the search API (requires user token).
ClickUp and GitHub data come from their respective CLIs if installed.
Sections are gracefully skipped if the CLI isn't available.

```
slackbuzz digest [flags]
```

### Examples

```
  # Full briefing
  slackbuzz digest

  # Only Slack
  slackbuzz digest --slack-only

  # Only GitHub
  slackbuzz digest --github-only

  # Since yesterday
  slackbuzz digest --since 1d

  # Output as JSON
  slackbuzz digest --json
```

### Options

```
      --clickup-only      Only show ClickUp tasks
      --github-only       Only show GitHub PRs
  -h, --help              help for digest
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --since string      Time window for activity (2h, 1d, 7d, 2w, or YYYY-MM-DD) (default "1d")
      --slack-only        Only show Slack activity
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

