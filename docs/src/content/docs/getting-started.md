---
title: Getting started
description: Initial setup walkthrough for the slackbuzz CLI.
---

This guide walks through initial setup and your first interaction with the CLI.

## Step 1: Create a Slack app

The easiest way to get started is to create a Slack app with all required scopes pre-configured:

```sh
slackbuzz app create
```

Paste an App Configuration Token when asked (from [api.slack.com/apps](https://api.slack.com/apps) → *Your App Configuration Tokens* → *Generate Token*). slackbuzz creates the app, then opens Slack in your browser — click **Allow** and both the bot and user tokens are stored automatically. Nothing else to copy.

No config token? Create the app in the browser instead — Slack's app creator opens pre-filled, and you paste back the Client ID and Secret:

```sh
slackbuzz app create --web
```

No terminal prompt available (e.g. running from an AI agent's `!` shell)? Copy the config token, then:

```sh
slackbuzz app create --clipboard
```

### Alternative: log in with existing tokens

If you already have bot and user tokens from an existing Slack app:

```sh
slackbuzz auth login
```

You will be prompted to paste a bot token (`xoxb-...`) and optionally a user token (`xoxp-...`).

### CI / non-interactive mode

Pipe a token via stdin, or read the one you just copied:

```sh
echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
slackbuzz auth login --clipboard
```

`app create` and `app update` also accept the config token via `--with-token`, `--clipboard`, or `$SLACKBUZZ_CONFIG_TOKEN`, checked in that order before any saved token. A token from `$SLACKBUZZ_CONFIG_TOKEN` overrides the saved one and is never written to disk.

### Check authentication status

```sh
slackbuzz auth status
```

### Re-authenticating after a slackbuzz upgrade

If a slackbuzz release adds new scope requirements (a new command whose
Slack method needs a scope your existing app manifest doesn't grant),
push the latest manifest to your existing Slack app and re-auth in one
flow:

```sh
slackbuzz app update
```

This calls `apps.manifest.update` with the canonical manifest, opens the
reinstall page so you can approve the new scopes, then prompts for the
regenerated bot + user tokens.

The manifest is generated automatically from the slackbuzz source tree
(`make manifest-gen`), so the scope list is always exactly what the
commands actually need — never more, never stale.

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

## Next steps

- See the full [Command reference](/slackbuzz-cli/commands/) for all available commands and flags.
- Learn about [Cross-tool integration](/slackbuzz-cli/cross-tool-integration/) for ClickUp and GitHub enrichment in activity views.
- Read [Using with AI agents](/slackbuzz-cli/ai-agents/) for integrating the CLI with Claude Code, Copilot, and Cursor.
