---
title: Security
description: Credential storage, security best practices, and vulnerability reporting.
---

# Security

## Credential storage

The CLI stores your Slack tokens in the operating system keyring:

- **macOS**: Keychain Access
- **Linux**: Secret Service API (GNOME Keyring, KDE Wallet)
- **Windows**: Windows Credential Manager

If the system keyring is unavailable, the CLI falls back to a plaintext file at `~/.config/slackbuzz/auth.yml` with a warning.

Tokens are never logged to stdout or stored in the config file.

## Token types

SlackBuzz uses two types of Slack tokens. The CLI automatically selects the correct token for each command.

| Token | Prefix | Automatic usage |
|-------|--------|-----------------|
| **Bot token** | `xoxb-` | Reading channels/users, reactions, system notifications |
| **User token** | `xoxp-` | Sending messages (as you), search, DMs, saved items, status |

Both tokens are stored separately in the keyring. The `slackbuzz app create` command creates a Slack app pre-configured with all required scopes for both token types.

### Required scopes

**Bot (`xoxb-`):** `chat:write`, `channels:history`, `channels:read`, `emoji:read`, `groups:history`, `groups:read`, `im:history`, `im:read`, `mpim:history`, `mpim:read`, `reactions:read`, `reactions:write`, `users:read`

**User (`xoxp-`):** `chat:write`, `im:write`, `search:read`, `stars:read`, `stars:write`, `users.profile:read`, `users.profile:write`

## Best practices

1. **Use the system keyring**: The default storage method. Avoid overriding it unless necessary.
2. **Use `app create`**: Creates a Slack app with the minimum required scopes. Avoid using tokens with broader permissions than needed.
3. **CI environments**: Pass tokens via environment variables and stdin, not as command-line arguments which may appear in process lists:

   ```sh
   echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
   ```

4. **Don't commit tokens**: Never commit Slack tokens to source control. Use `.gitignore` and CI secrets.
5. **Keep updated**: Always use the latest version for security patches.
6. **Logout when done**: Remove stored credentials with `slackbuzz auth logout`.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Email security concerns to: **info@campermate.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fixes (optional)

### Response timeline

| Step | Timeframe |
|------|-----------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Resolution | Depends on severity, typically within 30 days |

We will credit reporters in release notes unless you prefer anonymity. We ask that you do not publicly disclose the issue until we have had time to address it.

## Supported versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |
