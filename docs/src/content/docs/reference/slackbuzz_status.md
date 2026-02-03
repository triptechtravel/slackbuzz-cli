---
title: "slackbuzz status"
description: "Auto-generated reference for slackbuzz status"
---

## slackbuzz status

Slack status management

### Synopsis

View, set, or clear your Slack status.

Without a subcommand, shows your current status.
Requires a user token (xoxp-) with users.profile:write scope.

```
slackbuzz status [flags]
```

### Examples

```
  # Show current status
  slackbuzz status

  # Set status
  slackbuzz status set "In a meeting" :calendar:

  # Set status with expiration
  slackbuzz status set "Reviewing PRs" :eyes: --until 2h

  # Clear status
  slackbuzz status clear
```

### Options

```
  -h, --help   help for status
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line
* [slackbuzz status clear](/slackbuzz-cli/reference/slackbuzz_status_clear/)	 - Clear your Slack status
* [slackbuzz status set](/slackbuzz-cli/reference/slackbuzz_status_set/)	 - Set your Slack status

