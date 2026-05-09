---
title: AI agents
description: Using the slackbuzz CLI with AI coding agents like Claude Code, GitHub Copilot, and Cursor.
---

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

# 8. Agent edits a message if needed
slackbuzz message edit #engineering 1706000000.000000 "Updated: fix deployed in PR #43"

# 9. Agent searches for relevant files
slackbuzz file search "API spec" --type pdf --json
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
slackbuzz activity --all --since 1d --json

# Check saved items that need follow-up
slackbuzz later list --json

# Check threads you're in
slackbuzz threads --since 1d --json
```

## Claude Code skill

Install the SlackBuzz CLI skill into Claude Code so it automatically uses the CLI for Slack operations.

### Quick install (no clone required)

```sh
mkdir -p ~/.claude/skills/slackbuzz-cli
curl -fsSL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/skills/slackbuzz-cli/SKILL.md \
  -o ~/.claude/skills/slackbuzz-cli/SKILL.md
```

### From a local clone

If you have the repo checked out, symlink the skill so it stays up to date with `git pull`:

```sh
make install-skill
```

### What the skill does

Once installed, Claude Code will automatically use `slackbuzz` commands when you ask it to interact with Slack. You can use natural language instead of remembering CLI flags:

- "Check my Slack inbox"
- "Send @alice a message about the deployment"
- "Search Slack for the API spec discussion"
- "Set my status to deep work for 2 hours"

### @mention resolution

`@name` mentions in message bodies are resolved automatically to Slack's `<@USERID>` format before posting. Agents can write natural `@username` mentions without manually looking up user IDs:

```sh
# Mentions are resolved automatically
slackbuzz message send '#engineering' '@alice @bob please review PR #42'

# Also works in notify
slackbuzz notify #releases --message '@team v2.0 is live'
```

Unrecognized names pass through as-is. Use `slackbuzz user list --json` to discover available usernames.

## Agent mode

When calling slackbuzz from an AI agent or automation, set `SLACKBUZZ_AGENT=1` for agent-friendly defaults:

```sh
SLACKBUZZ_AGENT=1 slackbuzz send '#stand-up' "Daily update from CI"
```

Agent mode changes:
- **Bot token for sending**: Uses the bot token by default to avoid user-token scope gaps. Messages will appear from the bot app. Omit `SLACKBUZZ_AGENT=1` to post as yourself.
- **No interactive prompts**: Errors immediately if message text is missing instead of waiting for stdin.
- **Structured errors**: Outputs JSON error objects to stderr with `error`, `type`, and `suggestion` fields.

## Diagnostics

Run `slackbuzz doctor` to verify both tokens are valid and have the required scopes before using the CLI in automated workflows:

```sh
slackbuzz doctor
```

## Tips

- Use `--json` output when you need the agent to parse data programmatically
- Use `slackbuzz activity --json` to discover what needs attention
- Use `slackbuzz message list --json` to give the agent context from channel discussions
- Use `slackbuzz activity --all --json` to give the agent full context across mentions, DMs, and threads
- The activity view includes channel IDs, timestamps, and permalinks that agents can use to take action
- Use `slackbuzz message edit` and `slackbuzz message delete` to correct mistakes
- Use `slackbuzz file search` to find shared files and documents
- Use `slackbuzz react remove` to undo reactions
- All commands are safe to run multiple times (reads are idempotent, sends create new messages)
- Combine with `clickup` and `gh` CLIs for full cross-tool agent workflows
