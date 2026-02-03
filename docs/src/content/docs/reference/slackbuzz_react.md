---
title: "slackbuzz react"
description: "Auto-generated reference for slackbuzz react"
---

## slackbuzz react

React to a message

### Synopsis

Add an emoji reaction to a message.

The emoji should be in :emoji: format (colons are stripped automatically).
Requires a bot token (xoxb-) with reactions:write scope.

```
slackbuzz react <channel> <timestamp> <emoji> [flags]
```

### Examples

```
  slackbuzz react #general 1706000000.000000 :eyes:
  slackbuzz react #general 1706000000.000000 :white_check_mark:
  slackbuzz react #general 1706000000.000000 thumbsup
```

### Options

```
  -h, --help   help for react
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

