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
- User mentions Slack channels, users, or message timestamps

## Authentication

```bash
slackbuzz app create     # Create a new Slack app with the required scopes
slackbuzz app update     # Push the latest scope set to an existing app + re-auth
slackbuzz auth login     # Log in with bot and/or user token (manual paste)
slackbuzz auth status    # Check auth status and capabilities
```

**`app update` flow** — use when slackbuzz gains new scope requirements
(e.g. a future command needs a scope the existing manifest doesn't grant).
Pushes the canonical manifest to your existing app via `apps.manifest.update`,
opens the install page so you can approve the new scopes, then prompts for
the regenerated bot + user tokens. The manifest is auto-derived from method
usage (`make manifest-gen` runs in CI), so the scope list never drifts from
what commands actually need.

Two token types are required. **The CLI automatically selects the right token for each command — you do not need to specify which to use.**

- **Bot token** (`xoxb-`): Used automatically for reading channels, listing users, reactions, and system notifications
- **User token** (`xoxp-`): Used automatically for sending messages (as the user), search, DMs, saved items, status, and profile

Messages always post as the authenticated user by default. Only use `--as-bot` if you specifically want a message to come from the bot app rather than the user.

## Messaging

### Send Messages

```bash
# Send to a channel (shortcut — same as 'message send')
slackbuzz send '#general' "Hello team!"

# Send to a channel (full form)
slackbuzz message send '#general' "Hello team!"

# Send a DM (auto-opens DM channel)
slackbuzz send @alice "Hey, quick question"
slackbuzz send U02P3QC5H24 "Direct message by user ID"

# Send as bot (default uses user token if available)
slackbuzz send '#general' "Bot message" --as-bot
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

## Token Defaults

The CLI automatically selects the correct token for each command. You do not need to think about bot vs user mode — just run the command.

| Command | Token | Rationale |
|---------|-------|-----------|
| `message send`, `edit`, `delete` | **User** | Posts as the authenticated user |
| `message list` | **Bot** | Reads channel history |
| `channel list`, `channel info` | **Bot** | Reads channel metadata |
| `user list`, `user info` | **Bot** | Reads user profiles |
| `react`, `react remove` | **Bot** | Reactions via bot |
| `notify` | **Bot** | System/automated notifications |
| `thread link` | **Bot** | Generates permalinks |
| `activity`, `threads` | **User** | Slack search API (user-only) |
| `dm list` | **User** | Slack search API (user-only) |
| `message search`, `file search` | **User** | Slack search API (user-only) |
| `later list`, `add`, `remove` | **User** | Stars API (user-only) |
| `status`, `status set`, `clear` | **User** | Profile API (user-only) |

**Override:** Pass `--as-bot` on `message send`, `edit`, or `delete` to post as the bot app instead of the user. Only do this when explicitly requested.

**Missing permissions:** If a command fails due to a missing token or scope, the error message will indicate what's needed. Run `slackbuzz auth status` to check available capabilities.

## Common Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq <expr>` | Filter JSON with jq expression |
| `--template <tmpl>` | Format with Go template |
| `--as-bot` | Post as the bot app instead of the user (send/edit/delete only) |
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

The CLI automatically strips common shell escape artifacts from message text before sending. For example, zsh history expansion can turn exclamation marks into backslash-escaped versions when passed through double-quoted strings. The CLI detects and cleans these so the message arrives correctly in Slack.

## Fuzzy resolution

Channel and user names match in three tiers:

1. **Exact** (case-insensitive). `@michelle` → @michelle.
2. **Substring**. `@mich` → @michelle, `#stand` → #stand-up. Multiple
   matches resolve to the shortest (most-specific).
3. **Fuzzy** (edit distance, channel-only — never used for users to
   avoid mis-DMing the wrong person). `#stnd-p` → #stand-up.

Failing exact match shows "Did you mean: …?" suggestions in the error.

## Recent context

`slackbuzz dm` with no subcommand shows the most recently active DM
targets (newest first) and the commands to read/send them. Each
`message list`, `message send`, `notify` invocation records its target
into `~/.config/slack/recent.json` so future no-arg defaults flow.

## Key Behaviors

- **Automatic token selection**: The CLI picks the right token (bot or user) for each command. No need to specify — just run the command. Use `--as-bot` only when explicitly asked to post as the bot.
- **Self-DM via bot**: When sending a DM to yourself, the CLI automatically switches to the bot token so you receive a notification. The bot opens its own DM channel with you.
- **DM auto-detection**: `@user`, `U...` IDs, and bare names auto-resolve to DM channels (for the channel/target argument)
- **@mentions in message body**: Auto-resolved from `@name` to Slack's `<@USERID>` format before posting
- **First-name shorthand**: `@herman` resolves to `herman.gorbatovskii` or `Herman Gorbatovskii` when unambiguous
- **Shell unescape**: Common shell artifacts (backslash-escaped punctuation) are cleaned before sending
- **Case-insensitive resolution**: User lookup matches display names and usernames regardless of case
- **Permission feedback**: Missing tokens or scopes produce clear error messages. Use `slackbuzz auth status` to check capabilities.
- **Deeplinks**: Output includes clickable Slack deeplinks
## Formatting

Messages auto-convert standard Markdown to Slack mrkdwn:
- `**bold**` → `*bold*`
- `# Header` → `*Header*` (bold on its own line)
- `[text](url)` → `<url|text>`

```bash
# Auto-conversion happens by default
slackbuzz send '#releases' "## 🚀 v5.4.0\n- **New feature**: offline images"

# Use --blocks for Block Kit rendering (richer formatting, auto-splits long messages)
slackbuzz send '#releases' --blocks "## Release Notes\n- *Feature*: offline images"

# Use --raw to disable auto-conversion
slackbuzz send '#dev' --raw "**this stays as double asterisks**"
```

Formatting hints are shown on stderr when Markdown syntax is detected and converted.

## Agent Mode

When calling slackbuzz from an AI agent or automation, set `SLACKBUZZ_AGENT=1` to enable agent-friendly defaults:

```bash
SLACKBUZZ_AGENT=1 slackbuzz send '#stand-up' "Daily update from CI"
```

Agent mode changes:
- **Bot token for sending**: Uses the bot token by default to avoid user-token scope gaps (e.g. missing `channels:read`). Messages will appear from the bot app. Omit `SLACKBUZZ_AGENT=1` to post as yourself.
- **No interactive prompts**: Errors immediately if message text is missing instead of waiting for stdin
- **Structured errors**: Outputs JSON error objects to stderr with `error`, `type`, and `suggestion` fields

You can also run `slackbuzz doctor` to check that both tokens are valid and have the required scopes.

## Diagnostics

```bash
# Check token health and scope coverage
slackbuzz doctor

# Check auth status
slackbuzz auth status
```

## When commands fail with `missing_scope`

slackbuzz surfaces Slack's missing-scope responses with the specific scopes
needed and a one-line fix:

```
Error: chat.postMessage: missing_scope (needed scopes: chat:write)
```

Run `slackbuzz app update` to push the latest scope manifest to your existing
Slack app and re-auth. The manifest is auto-derived from the methods slackbuzz
calls, so the scope list is always exactly right — `app update` is the
canonical fix for any `missing_scope` error.

## Internal architecture (for context)

The CLI is generated from Slack's published OpenAPI 2.0 spec (committed to
`api/specs/slack_web.json`):

- **`make api-gen`** regenerates the typed Slack client (174 methods).
- **`make manifest-gen`** AST-walks `pkg/cmd/` for `slackapi.<Method>(...)`
  calls and writes the corresponding scope union to `pkg/cmd/app/manifest.go`.
- **`make verify-gen`** runs both and fails CI if generated artifacts have
  drifted — guarantees the manifest never lies about what the CLI actually
  needs.

Most surface area is generated; the hand-written pieces are:

- `internal/slackapi/client.go` — transport (`Do`, error envelope, sentinel errors)
- `internal/slackapi/blocks.go` — Block Kit composition helpers
- `internal/slackapi/files.go` — multipart `files.upload`
- `cmd/gen-api/patches.go` — narrow spec-quirk fixups (inline-vs-ref drift, nullable tuples)

See [Architecture](https://triptechtravel.github.io/slackbuzz-cli/architecture/) for the complete design.
