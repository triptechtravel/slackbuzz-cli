---
title: "slackbuzz app create"
description: "Auto-generated reference for slackbuzz app create"
---

## slackbuzz app create

Create a new Slack app with required scopes

### Synopsis

Create a new Slack app pre-configured with all scopes needed by the CLI,
install it, and store its tokens.

Two ways to create the app:

  Config token (default) — paste an App Configuration Token from
  api.slack.com/apps > Your App Configuration Tokens. The app is created
  via the manifest API.

  --web — no config token. Slack's "create from manifest" page opens
  pre-filled; click Create, then paste the Client ID and Client Secret.

Either way, Slack then opens in your browser: click Allow and both the bot
and user tokens are stored automatically. No token copying.

```
slackbuzz app create [flags]
```

### Examples

```
  # Interactive
  slackbuzz app create

  # Copy the config token, then (works without a terminal prompt)
  slackbuzz app create --clipboard

  # No config token: create the app in the browser
  slackbuzz app create --web

  # Scripted
  echo "xoxe.xoxp-..." | slackbuzz app create --with-token
  SLACKBUZZ_CONFIG_TOKEN=xoxe.xoxp-... slackbuzz app create
```

### Options

```
      --clipboard    Read config token from the system clipboard
  -h, --help         help for create
      --web          Create the app in the browser instead of with a config token
      --with-token   Read config token from stdin
```

### SEE ALSO

* [slackbuzz app](/slackbuzz-cli/reference/slackbuzz_app/)	 - Manage Slack app setup

