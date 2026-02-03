# @triptechtravel/slackbuzz-cli

A command-line tool for working with Slack messages, channels, and cross-tool workflows -- designed for developers who live in the terminal.

[![Release](https://img.shields.io/github/v/release/triptechtravel/slackbuzz-cli)](https://github.com/triptechtravel/slackbuzz-cli/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/triptechtravel/slackbuzz-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/triptechtravel/slackbuzz-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/triptechtravel/slackbuzz-cli)](https://goreportcard.com/report/github.com/triptechtravel/slackbuzz-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/triptechtravel/slackbuzz-cli.svg)](https://pkg.go.dev/github.com/triptechtravel/slackbuzz-cli)

`slackbuzz` bridges Slack, ClickUp, and GitHub from your terminal. Read your inbox, reply to messages, manage your status, and get a cross-tool morning briefing without opening a browser.

## Features

- **Activity inbox** -- see mentions, DMs, and threads that need your attention (alias: `inbox`)
- **Cross-tool digest** -- morning briefing combining Slack mentions, ClickUp tasks, and GitHub PRs
- **Message management** -- list, send, search, edit, and delete messages in any channel or DM
- **File search** -- search files shared across your workspace
- **Thread support** -- read and reply to threads
- **Reactions** -- add and remove reactions, see reactions inline in activity views
- **Saved items** -- save messages for later and manage your bookmarks
- **Status management** -- set, view, and clear your Slack status from the terminal
- **DM conversations** -- list and browse direct message conversations
- **Slack deeplinks** -- clickable links that open items directly in the Slack app
- **Emoji rendering** -- 1900+ emoji shortcodes rendered as Unicode, plus custom workspace emoji
- **ClickUp & GitHub enrichment** -- auto-detects `CU-` task IDs and GitHub PR/issue URLs in messages with actionable hints
- **AI-friendly** -- structured `--json` output and `--jq` filtering for AI coding agents (Claude Code, Copilot, Cursor)
- **JSON output** -- all list/view commands support `--json`, `--jq`, and `--template`
- **Shell completions** -- bash, zsh, fish, and PowerShell
- **Secure credentials** -- tokens stored in the system keyring with plaintext fallback

## Installation

### Go

```sh
go install github.com/triptechtravel/slackbuzz-cli/cmd/slackbuzz@latest
```

### Homebrew

```sh
brew install triptechtravel/tap/slackbuzz
```

### Binary releases

Download a prebuilt binary from the [releases page](https://github.com/triptechtravel/slackbuzz-cli/releases) and add it to your `PATH`.

## Quick start

Create a Slack app with all required scopes:

```sh
slackbuzz app create
```

This creates a Slack app, opens the install page, and prompts you to paste the OAuth tokens. Alternatively, log in with existing tokens:

```sh
slackbuzz auth login
```

Check your Slack inbox:

```sh
slackbuzz activity
```

Send a message:

```sh
slackbuzz message send #general "Hello from the terminal!"
```

Get a cross-tool morning briefing:

```sh
slackbuzz digest
```

## Command reference

### Inbox & activity

| Command | Description |
|---------|-------------|
| `slackbuzz activity` | Show mentions, DMs, and threads (alias: `inbox`) |
| `slackbuzz activity --dms` | Show direct messages |
| `slackbuzz activity --threads` | Show threads you're mentioned in |
| `slackbuzz activity --all --since 1d` | Everything from the last day |
| `slackbuzz threads` | Show threads you're participating in |
| `slackbuzz dm list` | List DM conversations with recent activity |
| `slackbuzz digest` | Cross-tool briefing (Slack + ClickUp + GitHub) |
| `slackbuzz later list` | Show saved/bookmarked messages |
| `slackbuzz later add <channel> <ts>` | Save a message for later |
| `slackbuzz later remove <channel> <ts>` | Unsave a message |

### Messaging

| Command | Description |
|---------|-------------|
| `slackbuzz message list <channel>` | Read channel or thread history |
| `slackbuzz message send <channel> "text"` | Send a message |
| `slackbuzz message edit <channel> <ts> "text"` | Edit a message |
| `slackbuzz message delete <channel> <ts>` | Delete a message |
| `slackbuzz message search <query>` | Search messages (user token required) |
| `slackbuzz file search <query>` | Search files (user token required) |
| `slackbuzz react <channel> <ts> :emoji:` | React to a message |
| `slackbuzz react remove <channel> <ts> :emoji:` | Remove a reaction |
| `slackbuzz notify <channel> "text"` | Send formatted Block Kit notifications |
| `slackbuzz thread <channel> <ts> "text"` | Reply to a thread |

### Channels & users

| Command | Description |
|---------|-------------|
| `slackbuzz channel list` | List channels |
| `slackbuzz channel info <channel>` | Show channel details |
| `slackbuzz user list` | List workspace members |
| `slackbuzz user info <user>` | Show user profile |

### Status

| Command | Description |
|---------|-------------|
| `slackbuzz status` | Show your current Slack status |
| `slackbuzz status set "text" :emoji:` | Set your status |
| `slackbuzz status set "text" :emoji: --until 2h` | Set status with expiry |
| `slackbuzz status clear` | Clear your status |

### Setup

| Command | Description |
|---------|-------------|
| `slackbuzz app create` | Create a Slack app with all required scopes |
| `slackbuzz auth login` | Authenticate with bot/user tokens |
| `slackbuzz auth logout` | Remove stored credentials |
| `slackbuzz auth status` | Show current authentication state |

### Utility

| Command | Description |
|---------|-------------|
| `slackbuzz version` | Print version, commit, and build date |
| `slackbuzz completion SHELL` | Generate shell completion scripts |

## Token types

SlackBuzz uses two types of Slack tokens:

| Token | Prefix | Used for |
|-------|--------|----------|
| **Bot token** | `xoxb-` | Sending messages, reading channels, reactions, emoji list |
| **User token** | `xoxp-` | Search, activity inbox, saved items, status management |

The `slackbuzz app create` command creates a Slack app pre-configured with all required scopes for both token types. After installing the app to your workspace, paste both tokens when prompted.

## Cross-tool integration

The `digest` command combines activity from Slack, ClickUp, and GitHub in a single view:

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

The digest shells out to the `clickup` and `gh` CLIs if installed. Sections are gracefully skipped if a CLI isn't available.

The activity view also auto-detects ClickUp task IDs (`CU-abc123`) and GitHub PR/issue URLs in messages and shows actionable hints:

```sh
$ slackbuzz activity

  @mention   alice        #engineering          2h ago
             Can you review CU-abc123? https://github.com/org/repo/pull/42
             -> Open in Slack
             -> clickup task view CU-abc123
             -> gh pr view 42
```

## Shell completions

Generate completion scripts for your shell:

```sh
# Bash
source <(slackbuzz completion bash)

# Zsh
source <(slackbuzz completion zsh)
# Or install permanently:
slackbuzz completion zsh > "${fpath[1]}/_slackbuzz"

# Fish
slackbuzz completion fish | source

# PowerShell
slackbuzz completion powershell | Out-String | Invoke-Expression
```

## CI usage

For non-interactive environments, pipe tokens via stdin:

```sh
echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
```

All list and view commands support `--json` for machine-readable output, and `--jq` for inline filtering:

```sh
slackbuzz activity --json
slackbuzz channel list --json --jq '.[].name'
slackbuzz message list #general --json
```

## Using with AI coding agents

The CLI is designed to work well with AI agents like Claude Code, GitHub Copilot, and Cursor. An AI agent can read Slack context, send messages, and manage tasks -- all without leaving the terminal.

```sh
# AI agent checks what needs attention
slackbuzz activity --json

# AI agent reads messages in a specific channel
slackbuzz message list #engineering --json --limit 10

# AI agent searches for relevant discussions
slackbuzz message search "deployment" --json

# AI agent replies to a thread
slackbuzz thread #engineering 1706000000.000000 "Fix deployed, see PR #42"

# AI agent reacts to acknowledge
slackbuzz react #engineering 1706000000.000000 :white_check_mark:

# AI agent gets cross-tool context
slackbuzz digest --json
```

The `--json` flag on all commands outputs structured data that agents can parse. The activity view includes channel IDs, timestamps, and permalinks that agents can use to take action.

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
