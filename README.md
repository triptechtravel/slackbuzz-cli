# @triptechtravel/slackbuzz-cli

A command-line tool for working with Slack messages, channels, and cross-tool workflows -- designed for developers who live in the terminal.

[![Release](https://img.shields.io/github/v/release/triptechtravel/slackbuzz-cli)](https://github.com/triptechtravel/slackbuzz-cli/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/triptechtravel/slackbuzz-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/triptechtravel/slackbuzz-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/triptechtravel/slackbuzz-cli)](https://goreportcard.com/report/github.com/triptechtravel/slackbuzz-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/triptechtravel/slackbuzz-cli.svg)](https://pkg.go.dev/github.com/triptechtravel/slackbuzz-cli)

`slackbuzz` brings Slack to your terminal. Read your inbox, reply to messages, manage your status, upload files, and search your workspace without opening a browser.

## Features

- **Activity inbox** -- see mentions, DMs, and threads that need your attention (alias: `inbox`)
- **Message management** -- list, send, search, edit, and delete messages in any channel or DM
- **File upload & search** -- upload files to channels/DMs (multi-file, thread support) and search shared files
- **Thread support** -- read and reply to threads
- **Reactions** -- add and remove reactions, see reactions inline in activity views
- **Saved items** -- save messages for later and manage your bookmarks
- **Status management** -- set, view, and clear your Slack status from the terminal
- **DM conversations** -- list and browse direct message conversations
- **Slack deeplinks** -- clickable links that open items directly in the Slack app
- **Emoji rendering** -- 1900+ emoji shortcodes rendered as Unicode, plus custom workspace emoji
- **@mention resolution** -- `@name` in message bodies auto-resolved to Slack's `<@USERID>` format, with first-name shorthand for dotted/multi-word names
- **Shell unescape** -- automatically strips shell escape artifacts (e.g. `\!` from zsh history expansion) before sending
- **ClickUp & GitHub enrichment** -- auto-detects `CU-` task IDs and GitHub PR/issue URLs in messages with actionable hints
- **AI-friendly** -- structured `--json` output and `--jq` filtering for AI coding agents (Claude Code, Copilot, Cursor)
- **JSON output** -- all list/view commands support `--json`, `--jq`, and `--template`
- **Shell completions** -- bash, zsh, fish, and PowerShell
- **Secure credentials** -- tokens stored in the system keyring with plaintext fallback
- **Fuzzy resolution** -- `@michell` → @michelle, `#stand` → #stand-up; "Did you mean: …?" suggestions on miss
- **Recent context** -- `slackbuzz dm` (no args) shows recently-active DMs; `message list/send` records targets for default-on-next-invoke
- **Spec-driven** -- 174 Slack methods generated from Slack's OpenAPI spec; manifest auto-derived from method usage so scope drift is structurally impossible
- **Self-updating manifest** -- `slackbuzz app update` pushes the latest scope set to your existing Slack app and walks through reinstall + new tokens

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

If slackbuzz later gains new scope requirements (e.g. a new method's
required scopes weren't in the original manifest), push the latest
manifest to your existing app and re-auth in one step:

```sh
slackbuzz app update
```

Check your Slack inbox:

```sh
slackbuzz activity
```

Send a message:

```sh
slackbuzz send '#general' "Hello from the terminal!"
```

Upload a file:

```sh
slackbuzz file upload report.pdf '#general'
```

## Team setup

Get a teammate from zero to fully operational with one command:

```sh
curl -sL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/scripts/setup.sh | bash
```

The script installs slackbuzz, sets up the Claude Code skill, and walks through authentication interactively. See the [team setup guide](https://triptechtravel.github.io/slackbuzz-cli/team-setup/) for details.

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
| `slackbuzz later list` | Show saved/bookmarked messages |
| `slackbuzz later add <channel> <ts>` | Save a message for later |
| `slackbuzz later remove <channel> <ts>` | Unsave a message |

### Messaging

| Command | Description |
|---------|-------------|
| `slackbuzz send <channel\|user> "text"` | Send a message (shortcut) |
| `slackbuzz message list <channel>` | Read channel or thread history |
| `slackbuzz message send <channel> "text"` | Send a message |
| `slackbuzz message edit <channel> <ts> "text"` | Edit a message |
| `slackbuzz message delete <channel> <ts>` | Delete a message |
| `slackbuzz message search <query>` | Search messages (user token required) |
| `slackbuzz file upload <file>... <channel>` | Upload files to a channel or DM |
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
| `slackbuzz app update` | Push the latest scope manifest to an existing app and re-auth |
| `slackbuzz auth login` | Authenticate with bot/user tokens |
| `slackbuzz auth logout` | Remove stored credentials |
| `slackbuzz auth status` | Show current authentication state |

### Diagnostics & utility

| Command | Description |
|---------|-------------|
| `slackbuzz doctor` | Check token health and required scopes |
| `slackbuzz version` | Print version, commit, and build date |
| `slackbuzz completion SHELL` | Generate shell completion scripts |

## Token types

SlackBuzz uses two types of Slack tokens. **The CLI automatically selects the correct token for each command** — you don't need to specify which to use.

| Token | Prefix | Automatic usage |
|-------|--------|-----------------|
| **Bot token** | `xoxb-` | Reading channels/users, reactions, system notifications (`notify`) |
| **User token** | `xoxp-` | Sending messages (as you), search, DMs, saved items, status |

Messages post as the authenticated user by default. Pass `--as-bot` on `message send`, `edit`, or `delete` to post as the bot app instead. When sending a DM to yourself, the CLI automatically switches to the bot so you receive a notification.

The `slackbuzz app create` command creates a Slack app pre-configured with all required scopes for both token types. After installing the app to your workspace, paste both tokens when prompted.

### Required scopes

**Auto-derived.** The full scope list is generated from the Slack API methods slackbuzz actually calls — see `pkg/cmd/app/manifest.go` for the canonical list (regenerated by `make manifest-gen`). Adding a method to a command flows automatically into the next manifest update; CI's `make verify-gen` fails the PR if anyone forgets to regenerate.

If a slackbuzz release adds new scope requirements, run `slackbuzz app update` to push the new manifest to your existing app and re-authenticate in one flow.

## Cross-tool enrichment

The activity view auto-detects ClickUp task IDs (`CU-abc123`) and GitHub PR/issue URLs in messages and shows actionable hints:

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
```

The `--json` flag on all commands outputs structured data that agents can parse. The activity view includes channel IDs, timestamps, and permalinks that agents can use to take action.

## Releasing

Releases are handled automatically by GitHub Actions using [goreleaser](https://goreleaser.com/). To create a new release:

```sh
git tag v0.x.y
git push origin main --tags
```

The workflow builds binaries for all platforms, creates a GitHub release, and updates the Homebrew tap. **Do not run `goreleaser` locally** — it will conflict with the CI release.

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
