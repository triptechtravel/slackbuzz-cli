---
name: slackbuzz-cli
description: Slack CLI for messaging, DMs, search, reactions, and status. Use when the user needs to interact with Slack — sending messages, checking activity/inbox, searching messages, managing reactions, or setting status. Prefer this CLI over raw Slack API calls.
---

# SlackBuzz CLI (`slackbuzz`)

Use the `slackbuzz` CLI instead of raw Slack API calls. It handles authentication (bot + user tokens), user/channel resolution, DM channel opening, and cross-tool integrations automatically.

## When to Use

- User asks to send a Slack message or DM
- User wants to check Slack activity, inbox, or threads
- User needs to search Slack messages or files
- User wants to react to messages, set status, or manage saved items
- User asks for a morning briefing or digest across Slack/ClickUp/GitHub
- User mentions Slack channels, users, or message timestamps

## Authentication

```bash
slackbuzz app create     # Create Slack app with required scopes
slackbuzz auth login     # Log in with bot and/or user token
slackbuzz auth status    # Check auth status and capabilities
```

Two token types:
- **Bot token** (`xoxb-`): Channels, reactions, user lists, posting messages
- **User token** (`xoxp-`): Search, DMs, stars, status, profile

## Messaging

### Send Messages

```bash
# Send to a channel
slackbuzz message send #general "Hello team!"

# Send a DM (auto-opens DM channel)
slackbuzz message send @alice "Hey, quick question"
slackbuzz message send U02P3QC5H24 "Direct message by user ID"

# Send as bot (default uses user token if available)
slackbuzz message send #general "Bot message" --as-bot
```

DM auto-detection: If the target looks like a user (`@name`, `U...` ID, or bare name), the CLI automatically opens a DM conversation via `conversations.open`.

### Read Messages

```bash
# Channel history
slackbuzz message list #general
slackbuzz message list #general --limit 50

# Thread replies
slackbuzz message list #general --thread 1706000000.000000

# DM history
slackbuzz message list @alice
```

### Edit & Delete

```bash
slackbuzz message edit #general 1706000000.000000 "Updated text"
slackbuzz message delete #general 1706000000.000000
```

### Search

```bash
# Search messages (requires user token)
slackbuzz message search "deploy production"

# Search files
slackbuzz file search "architecture diagram"
```

## Inbox & Activity

```bash
# Mentions (default)
slackbuzz activity

# DMs
slackbuzz activity --dms

# Threads you're in
slackbuzz activity --threads
slackbuzz threads    # Shortcut

# Everything combined
slackbuzz activity --all --since 1d

# Filter by channel or sender
slackbuzz activity --channel #engineering --from @alice

# DM conversations list
slackbuzz dm list
```

Activity detects ClickUp task IDs and GitHub PR/issue URLs in messages and shows actionable hints.

## Cross-Tool Digest

```bash
# Full briefing: Slack + ClickUp + GitHub
slackbuzz digest

# Scoped briefings
slackbuzz digest --slack-only
slackbuzz digest --github-only
slackbuzz digest --clickup-only
slackbuzz digest --since 1d
```

Integrates with `clickup` and `gh` CLIs if installed. Gracefully skips unavailable tools.

## Reactions

```bash
# Add reaction
slackbuzz react #general 1706000000.000000 :eyes:
slackbuzz react #general 1706000000.000000 thumbsup

# Remove reaction
slackbuzz react remove #general 1706000000.000000 :eyes:
```

## Status

```bash
# View current status
slackbuzz status

# Set status with emoji and optional expiration
slackbuzz status set "In a meeting" :calendar:
slackbuzz status set "Reviewing PRs" :eyes: --until 2h

# Clear status
slackbuzz status clear
```

## Saved Items

```bash
# List saved messages
slackbuzz later list

# Save/unsave a message
slackbuzz later add #general 1706000000.000000
slackbuzz later remove #general 1706000000.000000
```

## Notifications

```bash
# Release announcement
slackbuzz notify #releases --release v1.0.0

# Task status update
slackbuzz notify #updates --task CU-abc123 --status "deployed"

# Custom message
slackbuzz notify #general --message "Maintenance window starting"
```

## Thread Linking

```bash
# Link a Slack thread to a ClickUp task
slackbuzz thread link #general 1706000000.000000 --task CU-abc123
```

## Channels & Users

```bash
# List channels
slackbuzz channel list

# Channel info
slackbuzz channel info #general

# List users
slackbuzz user list

# User profile
slackbuzz user info @alice
```

## Common Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq <expr>` | Filter JSON with jq expression |
| `--template <tmpl>` | Format with Go template |
| `--as-bot` | Use bot token instead of user token (message send) |
| `--since <duration>` | Time filter (2h, 1d, 7d, 2w, or YYYY-MM-DD) |
| `--limit <n>` | Max results |

## @Mentioning Users in Messages

`@name` mentions in message bodies are **resolved automatically**. The CLI looks up usernames and display names (case-insensitive) and converts them to Slack's `<@USERID>` format before posting. Resolved mentions are confirmed on stderr.

```bash
# Mentions are resolved automatically — these will ping the users
slackbuzz message send '#channel' '@michelle @herman please review this'
# → stderr: Mentioning @michelle
# → stderr: Mentioning @herman

# Also works in notify --message
slackbuzz notify #general --message '@alice maintenance window starting'
```

### First-name resolution

When a user's Slack username is dotted (`herman.gorbatovskii`) or their display name has multiple words (`Herman Gorbatovskii`), the CLI also indexes the **first name** as a shorthand. Writing `@herman` will resolve to that user as long as the first name is unambiguous (only one user has that first name). If multiple users share a first name, use the full username or display name instead.

```bash
# These all resolve to the same user:
slackbuzz message send '#dev' '@herman.gorbatovskii check this'   # full username
slackbuzz message send '#dev' '@herman gorbatovskii check this'   # full display name
slackbuzz message send '#dev' '@herman, check this'               # first-name shorthand
```

Unrecognized names are left as-is (no error). Use `slackbuzz user list` to discover available usernames if a mention isn't resolving.

## Shell Escaping

The CLI automatically strips common shell escape artifacts from message text before sending. For example, zsh's history expansion can turn `!` into `\!` when passed through double-quoted strings. The CLI detects and cleans these so the message arrives correctly in Slack.

## Key Behaviors

- **DM auto-detection**: `@user`, `U...` IDs, and bare names auto-resolve to DM channels (for the channel/target argument)
- **@mentions in message body**: Auto-resolved from `@name` to Slack's `<@USERID>` format before posting
- **First-name shorthand**: `@herman` resolves to `herman.gorbatovskii` or `Herman Gorbatovskii` when unambiguous
- **Shell unescape**: Common shell artifacts like `\!` are cleaned before sending
- **Case-insensitive resolution**: User lookup matches display names and usernames regardless of case
- **Dual tokens**: Bot token for channel ops, user token for search/DMs/status
- **Deeplinks**: Output includes clickable Slack deeplinks
- **Cross-tool**: Digest combines Slack, ClickUp, and GitHub activity
