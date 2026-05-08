---
title: "slackbuzz dm"
description: "Auto-generated reference for slackbuzz dm"
---

## slackbuzz dm

Direct message management

### Synopsis

Direct-message management.

Reading and sending individual DMs works via:
  slackbuzz message list @user
  slackbuzz message send @user "text"

Running `slackbuzz dm` with no subcommand opens the most recent DM
conversation (per the local recents file).

```
slackbuzz dm [<user>] [flags]
```

### Options

```
  -h, --help   help for dm
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line
* [slackbuzz dm list](/slackbuzz-cli/reference/slackbuzz_dm_list/)	 - List DM conversations with recent activity

