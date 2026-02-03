---
title: "slackbuzz thread link"
description: "Auto-generated reference for slackbuzz thread link"
---

## slackbuzz thread link

Link a Slack thread to a ClickUp task

### Synopsis

Parse a Slack thread URL, extract the channel and timestamp,
and post a ClickUp task link as a reply to that thread.

The Slack URL format: https://workspace.slack.com/archives/CHANNEL_ID/pTIMESTAMP

```
slackbuzz thread link <slack-thread-url> <task-id> [flags]
```

### Examples

```
  slackbuzz thread link https://myteam.slack.com/archives/C012345/p1706000000000000 CU-abc123
```

### Options

```
  -h, --help              help for link
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz thread](/slackbuzz-cli/reference/slackbuzz_thread/)	 - Work with Slack threads

