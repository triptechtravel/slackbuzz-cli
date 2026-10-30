---
title: "slackbuzz app update"
description: "Auto-generated reference for slackbuzz app update"
---

## slackbuzz app update

Push the latest scope manifest and re-authenticate

### Synopsis

Update an existing Slack app's manifest with the CLI's current scope set,
then reinstall it and store new tokens. Use this after the CLI gains new
scope requirements (e.g. extending DM read support).

The command uses your saved App Configuration Token and App ID. If neither is
saved, you'll be prompted, or pass --app-id and supply the config token with
--clipboard, --with-token (stdin) or $SLACKBUZZ_CONFIG_TOKEN.

Reinstalling opens Slack in your browser — click Allow and the new tokens are
stored automatically. Apps made before this flow existed don't have their
Client ID and Secret saved; you'll be asked for them once (Basic Information >
App Credentials), or can press enter to paste tokens instead.

```
slackbuzz app update [flags]
```

### Examples

```
  # Interactive (uses saved app ID + config token)
  slackbuzz app update

  # Specify the app ID explicitly
  slackbuzz app update --app-id A0123456789

  # Copy the config token, then
  slackbuzz app update --clipboard

  # CI / scripted (config token via stdin)
  echo "xoxe.xoxp-..." | slackbuzz app update --with-token --app-id A0123456789
```

### Options

```
      --app-id app create   Slack app ID to update (defaults to saved value from app create)
      --clipboard           Read App Configuration Token from the system clipboard
  -h, --help                help for update
      --skip-reinstall      Skip opening the reinstall page and the new-token prompts (manifest push only)
      --with-token          Read App Configuration Token from stdin
```

### SEE ALSO

* [slackbuzz app](/slackbuzz-cli/reference/slackbuzz_app/)	 - Manage Slack app setup

