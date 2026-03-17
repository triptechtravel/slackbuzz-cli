---
title: Cross-tool integration
description: How slackbuzz bridges Slack, ClickUp, and GitHub from the terminal.
---

# Cross-tool integration

SlackBuzz is designed as the glue between Slack, ClickUp, and GitHub CLIs. It detects references to other tools in your Slack messages and provides actionable hints.

## Enrichment in activity views

The `activity` and `threads` commands auto-detect cross-tool references in message text:

### ClickUp task IDs

The regex `CU-[a-z0-9]+` is detected in message bodies. When found, slackbuzz shows an actionable hint:

```
  @mention   bob      #backend       5h ago
             CU-abc123 needs your input on the API design
             -> clickup task view CU-abc123
```

### GitHub PR/issue URLs

URLs matching `github.com/.../pull/\d+` or `github.com/.../issues/\d+` are detected:

```
  @mention   alice    #engineering   2h ago
             Can you review? https://github.com/org/repo/pull/42
             -> gh pr view 42
```

## Slack deeplinks

All activity and message views include clickable deeplinks that open items directly in the Slack desktop app. In terminals that support OSC 8 hyperlinks, timestamps and channel names are clickable.

Deeplink format: `slack://channel?team=TEAM_ID&id=CHANNEL_ID&message=TIMESTAMP`

## Workflow example

A typical morning workflow combining all three tools:

```sh
# 1. Check what needs attention
slackbuzz activity

# 2. Read the thread that mentions a ClickUp task
slackbuzz message list #backend --thread-ts 1706000000.000000

# 3. Check the ClickUp task
clickup task view CU-abc123

# 4. Reply in Slack
slackbuzz message send #backend "On it, updating the API now" --thread-ts 1706000000.000000

# 5. Update the ClickUp task status
clickup status set "in progress"

# 6. React to acknowledge
slackbuzz react #backend 1706000000.000000 :eyes:

# 7. Save a message for later follow-up
slackbuzz later add #backend 1706000000.000000
```

## Related CLIs

| Tool | Install | Used for |
|------|---------|----------|
| **slackbuzz** | `brew install triptechtravel/tap/slackbuzz` | Core Slack operations |
| **clickup** | `brew install triptechtravel/tap/clickup` | Follow up on ClickUp task ID enrichment hints |
| **gh** | `brew install gh` | Follow up on GitHub PR URL enrichment hints |

All three CLIs store credentials in the system keyring and support `--json` output for programmatic access.
