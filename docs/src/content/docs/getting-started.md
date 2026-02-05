---
title: Getting started
description: Initial setup walkthrough for the slackbuzz CLI.
---

# Getting started

This guide walks through initial setup and your first interaction with the CLI.

## Step 1: Create a Slack app

The easiest way to get started is to create a Slack app with all required scopes pre-configured:

```sh
slackbuzz app create
```

This creates a Slack app manifest, opens the Slack app creation page, and prompts you to paste the OAuth tokens after installing the app to your workspace.

### Alternative: log in with existing tokens

If you already have bot and user tokens from an existing Slack app:

```sh
slackbuzz auth login
```

You will be prompted to paste a bot token (`xoxb-...`) and optionally a user token (`xoxp-...`).

### CI / non-interactive mode

Pipe a token via stdin:

```sh
echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
```

### Check authentication status

```sh
slackbuzz auth status
```

## Step 2: Check your inbox

See what needs your attention -- mentions, DMs, and threads:

```sh
slackbuzz activity
```

Filter by type:

```sh
slackbuzz activity --dms         # Direct messages only
slackbuzz activity --threads     # Threads you're mentioned in
slackbuzz activity --all         # Everything combined
```

The activity view auto-detects ClickUp task IDs and GitHub PR URLs in messages and shows actionable hints.

## Step 3: Verify setup

Run the doctor to check that your tokens are valid and have the required scopes:

```sh
slackbuzz doctor
```

## Step 4: Send a message

```sh
slackbuzz send '#general' "Hello from the terminal!"
```

Reply to a thread:

```sh
slackbuzz thread #general 1706000000.000000 "Looks good, merging now"
```

React to a message:

```sh
slackbuzz react #general 1706000000.000000 :white_check_mark:
```

Edit or delete a message:

```sh
slackbuzz message edit #general 1706000000.000000 "corrected text"
slackbuzz message delete #general 1706000000.000000 --confirm
```

## Step 5: Set your status

```sh
slackbuzz status set "Coding" :computer: --until 2h
```

View your current status:

```sh
slackbuzz status
```

Clear it:

```sh
slackbuzz status clear
```

## Step 6: Get a cross-tool briefing

If you have the ClickUp CLI and GitHub CLI installed, get a combined view of everything across tools:

```sh
slackbuzz digest
```

This shows Slack mentions, ClickUp tasks assigned to you, and GitHub PRs needing your review.

## Next steps

- See the full [Command reference](/slackbuzz-cli/commands/) for all available commands and flags.
- Learn about [Cross-tool integration](/slackbuzz-cli/cross-tool-integration/) to combine Slack, ClickUp, and GitHub workflows.
- Read [Using with AI agents](/slackbuzz-cli/ai-agents/) for integrating the CLI with Claude Code, Copilot, and Cursor.
