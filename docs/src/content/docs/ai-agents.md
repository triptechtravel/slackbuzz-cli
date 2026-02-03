---
title: AI agents
description: Using the slackbuzz CLI with AI coding agents like Claude Code, GitHub Copilot, and Cursor.
---

# Using with AI agents

The CLI is designed to work with AI coding agents like Claude Code, GitHub Copilot, and Cursor. Every command supports structured output, so an agent can read Slack context, send messages, and manage tasks -- all without leaving the terminal.

## Why this matters

AI agents work best when they have full context. Instead of asking a developer to copy-paste Slack messages into a chat window, the agent can pull context directly from Slack, understand team discussions, and take action.

## Typical AI workflow

```sh
# 1. Agent checks what needs attention
slackbuzz activity --json

# 2. Agent reads messages in a specific channel
slackbuzz message list #engineering --json --limit 10

# 3. Agent searches for relevant discussions
slackbuzz message search "deployment" --json

# 4. Agent reads a specific thread for context
slackbuzz message list #engineering --thread-ts 1706000000.000000 --json

# 5. Agent implements the fix...

# 6. Agent replies to the thread
slackbuzz thread #engineering 1706000000.000000 "Fix deployed, see PR #42"

# 7. Agent reacts to acknowledge
slackbuzz react #engineering 1706000000.000000 :white_check_mark:
```

## Key features for AI agents

### Structured JSON output

All list and view commands support `--json` for machine-readable output:

```sh
# Get activity as JSON
slackbuzz activity --json

# Get channel list
slackbuzz channel list --json

# Filter with jq expressions
slackbuzz activity --json --jq '[.[] | select(.type == "mention")]'
```

### Cross-tool context

The `digest` command gives agents a complete picture across Slack, ClickUp, and GitHub:

```sh
# Get cross-tool context in one call
slackbuzz digest --json

# Slack mentions only
slackbuzz digest --slack-only --json
```

### ClickUp and GitHub enrichment

Activity views auto-detect ClickUp task IDs and GitHub PR URLs. The JSON output includes these references, allowing agents to follow up with the appropriate CLI:

```sh
# Agent reads activity, finds CU-abc123 and PR #42 references
slackbuzz activity --json

# Agent can then check the ClickUp task
clickup task view CU-abc123 --json

# And the GitHub PR
gh pr view 42 --json title,body,reviews
```

### Status management

Agents can manage the developer's Slack status as part of a workflow:

```sh
# Set status while working on a task
slackbuzz status set "Deep work" :headphones: --until 2h

# Clear when done
slackbuzz status clear
```

### Saved items

Agents can save important messages for later follow-up:

```sh
# Save a message for later
slackbuzz later add #engineering 1706000000.000000

# Check saved items
slackbuzz later list --json
```

## Example: Claude Code integration

When using Claude Code, the agent can be instructed to use the CLI as part of its workflow:

```
Task: Respond to the Slack thread about the API bug

1. Run `slackbuzz activity --json` to find the thread
2. Run `slackbuzz message list #backend --thread-ts <ts> --json` to read the discussion
3. Investigate and fix the bug
4. Run `slackbuzz thread #backend <ts> "Fixed in PR #42, the issue was..."`
5. Run `slackbuzz react #backend <ts> :white_check_mark:`
```

The CLI handles authentication via the system keyring, so no tokens need to be passed in prompts.

## Example: Morning standup automation

An agent can prepare a morning standup summary:

```sh
# Get everything from the last day
slackbuzz digest --since 1d --json

# Check saved items that need follow-up
slackbuzz later list --json

# Check threads you're in
slackbuzz threads --since 1d --json
```

## Tips

- Use `--json` output when you need the agent to parse data programmatically
- Use `slackbuzz activity --json` to discover what needs attention
- Use `slackbuzz message list --json` to give the agent context from channel discussions
- Use `slackbuzz digest --json` to give the agent cross-tool context
- The activity view includes channel IDs, timestamps, and permalinks that agents can use to take action
- All commands are safe to run multiple times (reads are idempotent, sends create new messages)
- Combine with `clickup` and `gh` CLIs for full cross-tool agent workflows
