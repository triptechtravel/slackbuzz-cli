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

Use "react remove" to remove a reaction.

```
slackbuzz react [<channel> <timestamp> <emoji>] [flags]
```

### Examples

```
  slackbuzz react #general 1706000000.000000 :eyes:
  slackbuzz react #general 1706000000.000000 :white_check_mark:
  slackbuzz react #general 1706000000.000000 thumbsup
  slackbuzz react remove #general 1706000000.000000 :eyes:
```

### Options

```
  -h, --help   help for react
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line
* [slackbuzz react remove](/slackbuzz-cli/reference/slackbuzz_react_remove/)	 - Remove a reaction from a message

