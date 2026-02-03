---
title: "slackbuzz auth login"
description: "Auto-generated reference for slackbuzz auth login"
---

## slackbuzz auth login

Authenticate with Slack

### Synopsis

Store Slack API tokens for CLI access.

Slack uses two types of tokens:
  Bot token (xoxb-)  — posting, reading channels, users
  User token (xoxp-) — searching messages (search:read scope)

The token type is auto-detected from its prefix. You can paste either
type and it will be stored correctly.

Get tokens from your Slack App: api.slack.com/apps > OAuth & Permissions.

In non-interactive environments (CI), pipe a token via stdin:
  echo "xoxb-..." | slackbuzz auth login --with-token

```
slackbuzz auth login [flags]
```

### Examples

```
  # Interactive (auto-detects token type)
  slackbuzz auth login

  # Hint which type you're providing
  slackbuzz auth login --bot-token
  slackbuzz auth login --user-token

  # Pipe token for CI
  echo "xoxb-..." | slackbuzz auth login --with-token
```

### Options

```
      --bot-token    Hint: storing a bot token (xoxb-)
  -h, --help         help for login
      --user-token   Hint: storing a user token (xoxp-)
      --with-token   Read token from standard input (for CI)
```

### SEE ALSO

* [slackbuzz auth](/slackbuzz-cli/reference/slackbuzz_auth/)	 - Authenticate with Slack

