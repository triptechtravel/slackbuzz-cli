---
title: Command reference
description: Complete reference for all slackbuzz CLI commands and flags.
---

# Command reference

All commands are invoked as subcommands of `slackbuzz`. Run `slackbuzz --help` for a summary, or `slackbuzz <command> --help` for details on any command.

---

## activity

Show what needs your attention -- mentions, DMs, and threads. Alias: `inbox`.

### `activity`

Search your Slack workspace for mentions, DMs, and threads using the `search.messages` API (user token required). Detects ClickUp task IDs and GitHub PR URLs in messages and shows actionable hints.

```sh
# Show recent mentions (default)
slackbuzz activity

# Show direct messages
slackbuzz activity --dms

# Show threads you're mentioned in
slackbuzz activity --threads

# Everything from the last day
slackbuzz activity --all --since 1d

# Filter by channel or sender
slackbuzz activity --channel #engineering --from @alice

# JSON output
slackbuzz activity --json
```

| Flag | Description |
|------|-------------|
| `--dms` | Show direct messages |
| `--threads` | Show threads you're mentioned in |
| `--all` | Show mentions, DMs, and threads combined |
| `--since DURATION` | How far back to search (e.g. `2h`, `1d`, `7d`, `2w`, or `YYYY-MM-DD`) |
| `--channel CHANNEL` | Filter to a specific channel |
| `--from USER` | Filter to a specific sender |
| `--limit N` | Maximum number of results |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## threads

### `threads`

Show threads you're participating in. Finds threads where you've posted using the search API.

```sh
# Recent threads
slackbuzz threads

# Threads from the last day
slackbuzz threads --since 1d

# JSON output
slackbuzz threads --json
```

| Flag | Description |
|------|-------------|
| `--since DURATION` | How far back to search (e.g. `2h`, `1d`, `7d`) |
| `--limit N` | Maximum number of results |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## dm

Manage direct message conversations.

### `dm list`

List DM conversations with recent activity, grouped by conversation partner.

```sh
slackbuzz dm list
slackbuzz dm list --since 1d
slackbuzz dm list --json
```

| Flag | Description |
|------|-------------|
| `--since DURATION` | How far back to search |
| `--limit N` | Maximum number of conversations |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

Each conversation is shown with the user, message count, and relative time on the first line, followed by a truncated preview of the last message. A quick-actions footer suggests next steps:

```
  @sarah         3 messages  2 hours ago
    last: "truncated message preview..."

---
Quick actions:
  Read:   slackbuzz message list @<user>
  Reply:  slackbuzz message send @<user> "text"
  Save:   slackbuzz later add @<user> <ts>
```

---

## later

Save messages for follow-up (Slack's "Later" / starred items).

### `later list`

Show your saved/bookmarked messages.

```sh
slackbuzz later list
slackbuzz later list --json
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `later add <channel> <timestamp>`

Save a message for later.

```sh
slackbuzz later add #general 1706000000.000000
```

### `later remove <channel> <timestamp>`

Remove a saved message.

```sh
slackbuzz later remove #general 1706000000.000000
```

---

## digest

### `digest`

Cross-tool morning briefing combining Slack mentions, ClickUp tasks, and GitHub PRs.

```sh
# Full briefing
slackbuzz digest

# Only Slack section
slackbuzz digest --slack-only

# Only ClickUp section
slackbuzz digest --clickup-only

# Only GitHub section
slackbuzz digest --github-only

# JSON output
slackbuzz digest --json
```

| Flag | Description |
|------|-------------|
| `--since DURATION` | How far back to search for Slack mentions |
| `--slack-only` | Only show Slack section |
| `--clickup-only` | Only show ClickUp section |
| `--github-only` | Only show GitHub section |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

Requires the [ClickUp CLI](https://github.com/triptechtravel/clickup-cli) and [GitHub CLI](https://cli.github.com/) for the respective sections. Missing CLIs are gracefully skipped.

---

## message

Read, send, and search Slack messages.

### `message list <channel>`

Read message history from a channel, DM, or thread.

```sh
# Read channel history
slackbuzz message list #general

# Read a DM conversation
slackbuzz message list @sarah

# Read a thread
slackbuzz message list #general --thread-ts 1706000000.000000

# Limit results
slackbuzz message list #general --limit 10
```

| Flag | Description |
|------|-------------|
| `--thread-ts TIMESTAMP` | Read a specific thread |
| `--limit N` | Maximum number of messages (default 20) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

Output includes a quick-actions footer with contextual commands:

```
---
Quick actions:
  Reply:  slackbuzz message send #general "text" --thread-ts <ts>
  React:  slackbuzz react #general <ts> :emoji:
  Edit:   slackbuzz message edit #general <ts> "new text"
```

### `message send <channel|user> <text>`

Send a message to a channel or DM. The first argument accepts a `#channel-name`, channel ID, `@username`, username, or user ID. When the target looks like a user, the CLI automatically opens a DM conversation via `conversations.open`.

**DM sending requires `im:write` and `chat:write` user token scopes.** Apps created with `slackbuzz app create` include these scopes by default. For existing apps, add the scopes at api.slack.com/apps > OAuth & Permissions.

```sh
# Send to a channel
slackbuzz message send #general "Hello from the terminal!"

# Send a DM by @username
slackbuzz message send @sarah "Quick question about the API"

# Send a DM by username (no @ needed)
slackbuzz message send herman "Hey, got a minute?"

# Send a DM by user ID
slackbuzz message send U02P3QC5H24 "Direct message by ID"

# Reply in a thread
slackbuzz message send #general "Fixed!" --thread-ts 1706000000.000000
```

| Flag | Description |
|------|-------------|
| `--thread-ts TIMESTAMP` | Send as a thread reply |
| `--as-bot` | Send as the bot instead of your user account |

### `message edit <channel> <timestamp> [new-text]`

Edit an existing message. If new-text is omitted, reads from stdin.

```sh
slackbuzz message edit #general 1706000000.000000 "updated text"
echo "corrected text" | slackbuzz message edit #general 1706000000.000000
```

### `message delete <channel> <timestamp>`

Delete a message. Prompts for confirmation in interactive mode.

```sh
slackbuzz message delete #general 1706000000.000000
slackbuzz message delete #general 1706000000.000000 --confirm
```

| Flag | Description |
|------|-------------|
| `--confirm` | Skip confirmation prompt (required in non-interactive mode) |

### `message search <query>`

Search messages across your workspace (user token required).

```sh
slackbuzz message search "deployment"
slackbuzz message search "error" --channel #general
slackbuzz message search "fix" --from @alice --since 2026-01-01
slackbuzz message search "deploy" --has :rocket:
slackbuzz message search "bug" --is thread --during week
slackbuzz message search "deploy" --sort score --page 2
```

| Flag | Description |
|------|-------------|
| `--channel CHANNEL` | Filter by channel |
| `--from USER` | Filter by user |
| `--since DATE` | Only show messages after this date (YYYY-MM-DD) |
| `--until DATE` | Only show messages before this date (YYYY-MM-DD) |
| `--has EMOJI` | Filter by emoji reaction (e.g. `:rocket:`) |
| `--is TYPE` | Filter by type: `dm`, `thread`, or `starred` |
| `--during PERIOD` | Filter by time period: `today`, `yesterday`, `week`, or `month` |
| `--sort ORDER` | Sort order: `timestamp` (default) or `score` |
| `--page N` | Page number for pagination |
| `--limit N` | Maximum number of results per page |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

Results include pagination info and a quick-actions footer:

```
---
Quick actions:
  Reply:  slackbuzz message send <channel> "text" --thread-ts <ts>
  React:  slackbuzz react <channel> <ts> :emoji:
  Save:   slackbuzz later add <channel> <ts>
```

---

## file

Search files shared in Slack.

### `file search <query>`

Search files across your workspace (user token required).

```sh
slackbuzz file search "design spec"
slackbuzz file search "report" --type pdf
slackbuzz file search "report" --channel #engineering
slackbuzz file search "diagram" --from @alice
slackbuzz file search "report" --page 2
```

| Flag | Description |
|------|-------------|
| `--channel CHANNEL` | Filter by channel |
| `--from USER` | Filter by uploader |
| `--type FILETYPE` | Filter by file type (e.g. `pdf`, `png`, `zip`) |
| `--limit N` | Maximum number of results per page |
| `--page N` | Page number for pagination |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## react

### `react <channel> <timestamp> <emoji>`

React to a message with an emoji.

```sh
slackbuzz react #general 1706000000.000000 :eyes:
slackbuzz react #general 1706000000.000000 :white_check_mark:
slackbuzz react #engineering 1706000000.000000 :thumbsup:
```

### `react remove <channel> <timestamp> <emoji>`

Remove a reaction from a message.

```sh
slackbuzz react remove #general 1706000000.000000 :eyes:
slackbuzz react remove #general 1706000000.000000 thumbsup
```

---

## thread

### `thread <channel> <timestamp> <text>`

Reply to a thread. Shorthand for `message send --thread-ts`.

```sh
slackbuzz thread #general 1706000000.000000 "Looks good, merging now"
```

### `thread link <channel> <timestamp>`

Generate a permalink for a thread.

```sh
slackbuzz thread link #general 1706000000.000000
```

---

## notify

### `notify <channel> <text>`

Send a formatted Block Kit notification to a channel.

```sh
slackbuzz notify #alerts "Deployment complete: v1.2.3 is live"
```

---

## channel

### `channel list`

List channels in the workspace.

```sh
slackbuzz channel list
slackbuzz channel list --json
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

Output includes a quick-actions footer:

```
---
Quick actions:
  Info:   slackbuzz channel info <channel>
  Read:   slackbuzz message list <channel>
  Send:   slackbuzz message send <channel> "text"
```

### `channel info <channel>`

Show details about a channel.

```sh
slackbuzz channel info #general
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## user

### `user list`

List workspace members.

```sh
slackbuzz user list
slackbuzz user list --json
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `user info <user>`

Show a user's profile.

```sh
slackbuzz user info @alice
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## status

Manage your Slack status.

### `status`

Show your current Slack status.

```sh
slackbuzz status
```

### `status set <text> <emoji>`

Set your Slack status with an emoji and optional expiry.

```sh
# Set indefinitely
slackbuzz status set "In a meeting" :calendar:

# Set with expiry
slackbuzz status set "Coding" :computer: --until 2h

# Set with a specific time
slackbuzz status set "OOO" :palm_tree: --until 2025-03-01
```

| Flag | Description |
|------|-------------|
| `--until DURATION` | Status expiry (e.g. `2h`, `1d`, or `YYYY-MM-DD`) |

### `status clear`

Clear your Slack status.

```sh
slackbuzz status clear
```

---

## auth

Manage authentication credentials.

### `auth login`

Authenticate with bot and user tokens. By default, prompts for tokens interactively. The tokens are validated against the Slack API and stored in the system keyring.

```sh
# Interactive token entry (default)
slackbuzz auth login

# Pipe token for CI
echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
```

| Flag | Description |
|------|-------------|
| `--with-token` | Read token from standard input (for CI environments) |

### `auth logout`

Remove stored credentials from the system keyring.

```sh
slackbuzz auth logout
```

### `auth status`

Display the current authentication state, including the workspace and user info.

```sh
slackbuzz auth status
```

---

## app

### `app create`

Create a Slack app with all required scopes for slackbuzz. Opens the Slack app creation page with a pre-configured manifest and prompts for the OAuth tokens.

```sh
slackbuzz app create
```

This is the recommended way to set up slackbuzz for the first time. The manifest includes all required bot and user token scopes.

---

## Utility commands

### `version`

Print the CLI version, commit SHA, and build date.

```sh
slackbuzz version
```

### `completion <SHELL>`

Generate shell completion scripts. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

```sh
slackbuzz completion bash
slackbuzz completion zsh
slackbuzz completion fish
slackbuzz completion powershell
```

See the [Installation](/slackbuzz-cli/installation/#shell-completions) page for setup instructions.
