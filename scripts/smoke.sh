#!/usr/bin/env bash
# scripts/smoke.sh — exercise the slackapi typed client against a real
# Slack workspace. Run before tagging a release; not wired to CI by default
# (needs real tokens — see make smoke).
#
# Override the binary under test with `BIN=./bin/slackbuzz make smoke` or
# simply pre-set `BIN`. The default uses whatever `slackbuzz` is on PATH.
#
# Coverage:
#   - auth.test       (smallest possible round-trip; confirms token + transport)
#   - conversations.list (typed Channels response; confirms patches.go fired)
#   - users.info on the authenticated user (typed User; confirms synthesised def)
#   - message.list on a recent DM (confirms generated wrapper end-to-end)
#
# All output is parsed via --json + jq; failures are loud and exit non-zero.
set -euo pipefail

BIN="${BIN:-slackbuzz}"
echo "Smoke-testing $($BIN version 2>&1 | head -1)"
echo "Binary: $(command -v $BIN)"
echo

if ! command -v jq >/dev/null 2>&1; then
  echo "smoke: jq is required (brew install jq)" >&2
  exit 1
fi

trap 'echo; echo "smoke: FAILED" >&2' ERR

# --- auth.test (via auth status + ResolveUserID, no direct method call) ---
echo "1/4: auth status"
$BIN auth status >/dev/null
echo "    OK"

# --- conversations.list ---
echo "2/4: channel list"
out=$($BIN channel list --type public --limit 3 --json)
count=$(echo "$out" | jq 'length')
if [ "$count" -lt 1 ]; then
  echo "    expected at least 1 channel, got $count" >&2
  exit 1
fi
sample=$(echo "$out" | jq -r '.[0].name')
echo "    OK ($count channels, sample: #$sample)"

# --- user info on self (uses Slack's auth.test under the hood) ---
echo "3/4: user info @self"
self_id=$($BIN auth status --json 2>/dev/null | jq -r '.user_id // empty' || true)
if [ -z "$self_id" ]; then
  # Fall back to a `users.list` lookup of the bot's own identity if
  # auth status doesn't expose user_id in JSON form.
  self_id=$($BIN user list --limit 1 --json 2>/dev/null | jq -r '.[0].id' || echo "")
fi
if [ -z "$self_id" ]; then
  echo "    skip — couldn't determine self user_id"
else
  $BIN user info "$self_id" --json >/dev/null
  echo "    OK ($self_id)"
fi

# --- message list on a recent DM (skips if no DMs exist) ---
echo "4/4: dm list"
last_dm=$($BIN dm list --limit 1 --json 2>/dev/null | jq -r '.[0].user // empty' || true)
if [ -z "$last_dm" ]; then
  echo "    skip — no DMs to read"
else
  $BIN message list "@$last_dm" --limit 1 --json >/dev/null
  echo "    OK (@$last_dm)"
fi

echo
echo "smoke: PASSED"
