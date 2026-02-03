---
title: "slackbuzz"
description: "Auto-generated reference for slackbuzz"
---

## slackbuzz

Slack CLI - message, search, and manage channels from the command line

### Synopsis

Work with Slack channels, messages, and users from your terminal.

Integrates with ClickUp and GitHub CLIs for cross-tool developer workflows.

GETTING STARTED
  slackbuzz app create          Create a Slack app with required scopes
  slackbuzz auth login           Log in with bot/user tokens

INBOX & ACTIVITY
  slackbuzz activity             See mentions, DMs, and threads (alias: inbox)
  slackbuzz threads              Threads you're participating in
  slackbuzz dm list              DM conversations with recent activity
  slackbuzz digest               Cross-tool briefing (Slack + ClickUp + GitHub)

MESSAGING
  slackbuzz message list <chan>  Read channel/thread history
  slackbuzz message send <chan>  Send a message
  slackbuzz message edit <ch> <ts>  Edit a message
  slackbuzz message delete <ch> <ts>  Delete a message
  slackbuzz message search <q>  Search messages (user token required)
  slackbuzz file search <q>     Search files (user token required)
  slackbuzz react <chan> <ts>    React to a message
  slackbuzz react remove <ch> <ts>  Remove a reaction

CHANNELS & USERS
  slackbuzz channel list         List channels
  slackbuzz user list            List workspace members

SAVED ITEMS & STATUS
  slackbuzz later list           Show saved/bookmarked messages
  slackbuzz status               View or set your Slack status

TIPS
  Most commands support --json for JSON output and --jq for filtering.
  Use --help on any command for full usage details.
  Deeplinks let you click to open items directly in Slack.

### Options

```
  -h, --help   help for slackbuzz
```

### SEE ALSO

* [slackbuzz activity](/slackbuzz-cli/reference/slackbuzz_activity/)	 - Show what needs your attention — mentions, DMs, threads
* [slackbuzz app](/slackbuzz-cli/reference/slackbuzz_app/)	 - Manage Slack app setup
* [slackbuzz auth](/slackbuzz-cli/reference/slackbuzz_auth/)	 - Authenticate with Slack
* [slackbuzz channel](/slackbuzz-cli/reference/slackbuzz_channel/)	 - Manage Slack channels
* [slackbuzz completion](/slackbuzz-cli/reference/slackbuzz_completion/)	 - Generate shell completion scripts
* [slackbuzz digest](/slackbuzz-cli/reference/slackbuzz_digest/)	 - Cross-tool morning briefing (Slack + ClickUp + GitHub)
* [slackbuzz dm](/slackbuzz-cli/reference/slackbuzz_dm/)	 - Direct message management
* [slackbuzz file](/slackbuzz-cli/reference/slackbuzz_file/)	 - Search and manage files
* [slackbuzz later](/slackbuzz-cli/reference/slackbuzz_later/)	 - Saved/bookmarked items
* [slackbuzz message](/slackbuzz-cli/reference/slackbuzz_message/)	 - Send and read Slack messages
* [slackbuzz notify](/slackbuzz-cli/reference/slackbuzz_notify/)	 - Send formatted notifications
* [slackbuzz react](/slackbuzz-cli/reference/slackbuzz_react/)	 - React to a message
* [slackbuzz status](/slackbuzz-cli/reference/slackbuzz_status/)	 - Slack status management
* [slackbuzz thread](/slackbuzz-cli/reference/slackbuzz_thread/)	 - Work with Slack threads
* [slackbuzz threads](/slackbuzz-cli/reference/slackbuzz_threads/)	 - Show threads you're participating in
* [slackbuzz user](/slackbuzz-cli/reference/slackbuzz_user/)	 - Manage Slack users
* [slackbuzz version](/slackbuzz-cli/reference/slackbuzz_version/)	 - Print the version of slackbuzz CLI

