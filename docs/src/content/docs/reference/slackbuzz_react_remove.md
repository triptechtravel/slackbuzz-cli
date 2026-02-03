---
title: "slackbuzz react remove"
description: "Auto-generated reference for slackbuzz react remove"
---

## slackbuzz react remove

Remove a reaction from a message

### Synopsis

Remove an emoji reaction from a message.

The emoji should be in :emoji: format (colons are stripped automatically).
Requires a bot token (xoxb-) with reactions:write scope.

```
slackbuzz react remove <channel> <timestamp> <emoji> [flags]
```

### Examples

```
  slackbuzz react remove #general 1706000000.000000 :eyes:
  slackbuzz react remove #general 1706000000.000000 thumbsup
```

### Options

```
  -h, --help   help for remove
```

### SEE ALSO

* [slackbuzz react](/slackbuzz-cli/reference/slackbuzz_react/)	 - React to a message

