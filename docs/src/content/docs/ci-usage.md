---
title: CI usage
description: Using the slackbuzz CLI in CI/CD pipelines with JSON output and non-interactive auth.
---

# CI usage

The CLI works in non-interactive environments like GitHub Actions, GitLab CI, and other CI/CD pipelines.

## Authentication

Pass your Slack bot token via stdin using `--with-token`:

```sh
echo "$SLACK_BOT_TOKEN" | slackbuzz auth login --with-token
```

Store the token as a CI secret (e.g., GitHub Actions secret, GitLab CI variable). Do not pass it as a command-line argument.

## JSON output

All list and view commands support `--json` for machine-readable output:

```sh
slackbuzz activity --json
slackbuzz channel list --json
slackbuzz message list #general --json
```

### Filtering with `--jq`

Use `--jq` to extract specific fields inline:

```sh
# Get channel names
slackbuzz channel list --json --jq '.[].name'

# Get recent activity as JSON
slackbuzz activity --json --jq '.[0].text'

# Filter messages
slackbuzz message list #general --json --jq '[.[] | select(.user == "alice")]'
```

## GitHub Actions examples

### Send a notification on deploy

```yaml
name: Deploy Notification
on:
  deployment_status:

jobs:
  notify:
    if: github.event.deployment_status.state == 'success'
    runs-on: ubuntu-latest
    steps:
      - name: Install slackbuzz
        run: |
          curl -sL https://github.com/triptechtravel/slackbuzz-cli/releases/latest/download/slackbuzz_linux_amd64.tar.gz | tar xz
          sudo mv slackbuzz /usr/local/bin/
      - name: Authenticate
        run: echo "${{ secrets.SLACK_BOT_TOKEN }}" | slackbuzz auth login --with-token
      - name: Notify channel
        run: slackbuzz message send "#deploys" "Deployed ${{ github.event.deployment.ref }} to ${{ github.event.deployment.environment }}"
```

### Post CI results

```yaml
name: CI Notification
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Install slackbuzz
        run: |
          curl -sL https://github.com/triptechtravel/slackbuzz-cli/releases/latest/download/slackbuzz_linux_amd64.tar.gz | tar xz
          sudo mv slackbuzz /usr/local/bin/
      - name: Authenticate
        run: echo "${{ secrets.SLACK_BOT_TOKEN }}" | slackbuzz auth login --with-token
      - name: Post result
        run: |
          CONCLUSION="${{ github.event.workflow_run.conclusion }}"
          BRANCH="${{ github.event.workflow_run.head_branch }}"
          slackbuzz message send "#ci" "CI ${CONCLUSION}: ${BRANCH}"
```

### Send Block Kit notifications

```yaml
- name: Notify with formatting
  run: slackbuzz notify "#alerts" "Build *passed* for \`${{ github.ref_name }}\` :white_check_mark:"
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (API failure, invalid input, etc.) |

Commands that produce no output (e.g., no messages found) still exit with code 0.
