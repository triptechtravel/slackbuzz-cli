---
title: Team Setup
description: Get your team up and running with slackbuzz in one command.
---

# Team setup

Get a teammate from zero to fully operational with a single command.

## One-line setup

```sh
curl -sL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/scripts/setup.sh | bash
```

The script walks through each step interactively: installing slackbuzz, setting up the Claude Code skill, and authenticating with Slack. It's idempotent — safe to run again if anything changes.

If you have the repo cloned locally, you can also run:

```sh
make setup
```

## What you'll need

| Prerequisite | Required? | Notes |
|---|---|---|
| **Homebrew** | Yes | The script installs slackbuzz via `brew`. Install from [brew.sh](https://brew.sh). |
| **Bot token** (`xoxb-...`) | Yes | Shared across the team. Your team lead will provide this. |
| **User token** (`xoxp-...`) | Yes | Unique per person. Each team member generates their own. |
| **Claude Code** | Optional | If installed, the script adds the slackbuzz skill automatically. |

## Understanding the tokens

slackbuzz uses two Slack tokens. The CLI selects the right one automatically — you never need to specify which to use.

**Bot token** (`xoxb-...`) — shared across the team. One app installation covers everyone. Used for reading channels, listing users, and reactions.

**User token** (`xoxp-...`) — unique per person. Each team member installs the Slack app to their own account. Used for sending messages as you, search, DMs, and status.

Both tokens come from the same Slack app. The bot token is the same for everyone on the team; the user token is different per person.

## Manual step-by-step

If you prefer not to use the setup script, here's what it does:

### 1. Install slackbuzz

```sh
brew install triptechtravel/tap/slackbuzz
```

### 2. Install the Claude Code skill (optional)

```sh
mkdir -p ~/.claude/skills/slackbuzz-cli
curl -fsSL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/skills/slackbuzz-cli/SKILL.md \
  -o ~/.claude/skills/slackbuzz-cli/SKILL.md
```

Or from a local clone:

```sh
make install-skill
```

### 3. Authenticate

```sh
slackbuzz auth login --bot-token
slackbuzz auth login --user-token
```

Paste each token when prompted. Get them from your Slack app's **OAuth & Permissions** page.

### 4. Verify

```sh
slackbuzz doctor
```

## For team leads: creating the Slack app

Before your team can authenticate, someone needs to create the Slack app and install it to the workspace.

### Create the app

The fastest way is to use the built-in command:

```sh
slackbuzz app create
```

This creates a Slack app pre-configured with all required scopes for both token types, opens the install page in your browser, and prompts you to paste the OAuth tokens.

### Share with your team

After the app is installed to your workspace:

1. Go to [api.slack.com/apps](https://api.slack.com/apps) and select the app
2. Navigate to **OAuth & Permissions**
3. Copy the **Bot User OAuth Token** (`xoxb-...`) — this is the same for everyone
4. Share the bot token with your team (e.g., in a private Slack channel or password manager)
5. Each team member visits the same OAuth page and copies their own **User OAuth Token** (`xoxp-...`)

Then each team member runs the setup script and pastes both tokens when prompted.

### Required scopes

The `slackbuzz app create` command sets these up automatically. If creating the app manually:

**Bot token (`xoxb-`):** `chat:write`, `channels:history`, `channels:read`, `emoji:read`, `groups:history`, `groups:read`, `im:history`, `im:read`, `im:write`, `mpim:history`, `mpim:read`, `reactions:read`, `reactions:write`, `users:read`

**User token (`xoxp-`):** `channels:read`, `chat:write`, `groups:read`, `im:read`, `im:write`, `mpim:read`, `search:read`, `stars:read`, `stars:write`, `users:read`, `users.profile:read`, `users.profile:write`

## Troubleshooting

### "Not authenticated" errors

Re-run authentication for the missing token:

```sh
slackbuzz auth status          # See which tokens are configured
slackbuzz auth login --bot-token   # Re-enter bot token
slackbuzz auth login --user-token  # Re-enter user token
```

### Scope errors ("missing_scope")

The Slack app needs additional OAuth scopes. Ask your team lead to:

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → your app → **OAuth & Permissions**
2. Add the missing scope listed in the error message
3. Reinstall the app to the workspace
4. Team members re-run `slackbuzz auth login` with the new tokens

### Updating slackbuzz

```sh
brew update && brew upgrade triptechtravel/tap/slackbuzz
```

Or re-run the setup script — it detects existing installations and upgrades automatically.

### Updating the Claude Code skill

```sh
curl -fsSL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/skills/slackbuzz-cli/SKILL.md \
  -o ~/.claude/skills/slackbuzz-cli/SKILL.md
```

If you installed via `make install-skill` (symlink), just `git pull` in the repo.

### Uninstalling

```sh
brew uninstall slackbuzz                       # Remove the CLI
rm -rf ~/.claude/skills/slackbuzz-cli          # Remove the Claude Code skill
slackbuzz auth logout                          # Remove stored tokens (run before uninstall)
```
