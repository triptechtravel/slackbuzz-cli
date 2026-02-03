---
title: Cross-tool integration
description: How slackbuzz bridges Slack, ClickUp, and GitHub from the terminal.
---

# Cross-tool integration

SlackBuzz is designed as the glue between Slack, ClickUp, and GitHub CLIs. It detects references to other tools in your Slack messages and provides actionable hints.

## The digest command

The `digest` command combines activity from all three tools in a single view:

```sh
$ slackbuzz digest

SLACK -- 3 unread mentions
  @alice in #dev        "Can you review PR #42?"          2h ago
  @bob in #backend      "CU-abc needs your input"         5h ago
  @sarah DM             "meeting at 3?"                   1d ago

CLICKUP -- 2 tasks assigned to you
  CU-abc123  In Review   API redesign
  CU-def456  In Progress Deploy pipeline fix

GITHUB -- 1 PR needs your review
  #42  feat: add caching layer    alice  +120 -30

---
Quick actions:
  slackbuzz message send #dev "reviewing now"
  clickup task update CU-abc123 --status complete
  gh pr review 42 --approve
```

### How it works

The digest shells out to the `clickup` and `gh` CLIs if installed:

- **Slack**: Uses `search.messages` API to find your recent mentions
- **ClickUp**: Runs `clickup task list --assignee me --json` to find your assigned tasks
- **GitHub**: Runs `gh pr list --review-requested=@me --json` to find PRs needing review

Sections are gracefully skipped if a CLI is not installed (`exec.LookPath`).

### Flags

```sh
slackbuzz digest --since 2d        # Look back 2 days instead of default
slackbuzz digest --slack-only      # Only show Slack section
slackbuzz digest --clickup-only    # Only show ClickUp section
slackbuzz digest --github-only     # Only show GitHub section
slackbuzz digest --json            # Structured output
```

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
# 1. Check what needs attention across all tools
slackbuzz digest

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

## Required CLIs

| Tool | Install | Used by |
|------|---------|---------|
| **slackbuzz** | `brew install triptechtravel/tap/slackbuzz` | Core Slack operations |
| **clickup** | `brew install triptechtravel/tap/clickup` | `digest` ClickUp section, task ID enrichment hints |
| **gh** | `brew install gh` | `digest` GitHub section, PR URL enrichment hints |

All three CLIs store credentials in the system keyring and support `--json` output for programmatic access.
