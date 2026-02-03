---
title: "slackbuzz app create"
description: "Auto-generated reference for slackbuzz app create"
---

## slackbuzz app create

Create a new Slack app with required scopes

### Synopsis

Create a new Slack app pre-configured with all scopes needed by the CLI.

This command uses the Slack apps.manifest API to create an app. You need
an App Configuration Token from: api.slack.com/apps > Your App Configuration Tokens.

After the app is created, it opens the install page in your browser.
Once installed, copy the Bot and User OAuth tokens for 'slackbuzz auth login'.

```
slackbuzz app create [flags]
```

### Examples

```
  # Interactive (prompts for config token)
  slackbuzz app create

  # Pipe config token for CI
  echo "xoxe.xoxp-..." | slackbuzz app create --with-token
```

### Options

```
  -h, --help         help for create
      --with-token   Read config token from stdin
```

### SEE ALSO

* [slackbuzz app](/slackbuzz-cli/reference/slackbuzz_app/)	 - Manage Slack app setup

