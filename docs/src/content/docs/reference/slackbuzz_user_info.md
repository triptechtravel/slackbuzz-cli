---
title: "slackbuzz user info"
description: "Auto-generated reference for slackbuzz user info"
---

## slackbuzz user info

Show user profile

### Synopsis

Display detailed information about a Slack user.

Accepts @username or a user ID.

```
slackbuzz user info <user> [flags]
```

### Examples

```
  slackbuzz user info @alice
  slackbuzz user info U012345 --json
```

### Options

```
  -h, --help              help for info
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [slackbuzz user](/slackbuzz-cli/reference/slackbuzz_user/)	 - Manage Slack users

