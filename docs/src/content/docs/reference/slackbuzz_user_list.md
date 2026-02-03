---
title: "slackbuzz user list"
description: "Auto-generated reference for slackbuzz user list"
---

## slackbuzz user list

List workspace users

### Synopsis

List all active users in the Slack workspace.

```
slackbuzz user list [flags]
```

### Examples

```
  # List active users
  slackbuzz user list

  # Include deactivated users
  slackbuzz user list --include-deactivated

  # Output as JSON
  slackbuzz user list --json
```

### Options

```
  -h, --help                  help for list
      --include-deactivated   Include deactivated users
      --jq string             Filter JSON output using a jq expression
      --json                  Output JSON
      --limit int             Maximum number of users to list (default 200)
      --template string       Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz user](/slackbuzz-cli/reference/slackbuzz_user/)	 - Manage Slack users

