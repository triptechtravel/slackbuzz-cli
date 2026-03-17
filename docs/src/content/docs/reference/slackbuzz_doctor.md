---
title: "slackbuzz doctor"
description: "Auto-generated reference for slackbuzz doctor"
---

## slackbuzz doctor

Check token health and required scopes

### Synopsis

Validate bot and user tokens, then probe key API scopes to detect
permission gaps before they cause errors.

Checks:
  - Bot token validity and channels:read scope
  - User token validity and channels:read scope
  - Reports pass/fail with remediation advice

Exit code 1 if any check fails.

```
slackbuzz doctor [flags]
```

### Examples

```
  slackbuzz doctor
```

### Options

```
  -h, --help   help for doctor
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

