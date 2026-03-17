#!/usr/bin/env bash
# slackbuzz team setup — run with:
#   curl -sL https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/scripts/setup.sh | bash
#
# Idempotent: safe to run multiple times.

set -euo pipefail

# ── Reconnect stdin when piped from curl ─────────────────────────────
# When run as `curl ... | bash`, stdin is the curl stream. Interactive
# prompts (e.g. slackbuzz auth login) need the real terminal.
if [ ! -t 0 ]; then
  exec </dev/tty
fi

# ── Color helpers (degrade gracefully for non-terminals) ─────────────
if [ -t 1 ] && command -v tput &>/dev/null && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  BOLD=$(tput bold)
  DIM=$(tput dim)
  RESET=$(tput sgr0)
  RED=$(tput setaf 1)
  GREEN=$(tput setaf 2)
  YELLOW=$(tput setaf 3)
  BLUE=$(tput setaf 4)
  CYAN=$(tput setaf 6)
else
  BOLD="" DIM="" RESET="" RED="" GREEN="" YELLOW="" BLUE="" CYAN=""
fi

info()    { printf "%s[info]%s  %s\n"    "$BLUE"   "$RESET" "$*"; }
success() { printf "%s[ok]%s    %s\n"    "$GREEN"  "$RESET" "$*"; }
warn()    { printf "%s[warn]%s  %s\n"    "$YELLOW" "$RESET" "$*"; }
error()   { printf "%s[error]%s %s\n"    "$RED"    "$RESET" "$*"; }
step()    { printf "\n%s━━ Step %s ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n" "$BOLD" "$1" "$RESET"; }

# ── Banner ───────────────────────────────────────────────────────────
printf "\n"
printf "%s  slackbuzz — team setup%s\n" "$BOLD" "$RESET"
printf "  Bridge Slack, ClickUp, and GitHub from your terminal.\n"
printf "  %shttps://triptechtravel.github.io/slackbuzz-cli%s\n" "$DIM" "$RESET"
printf "\n"

# ── Step 1: Prerequisites ───────────────────────────────────────────
step "1/6: Check prerequisites"

if ! command -v brew &>/dev/null; then
  error "Homebrew is required but not installed."
  info  "Install it from https://brew.sh then re-run this script."
  exit 1
fi
success "Homebrew found"

HAS_CLAUDE=false
if command -v claude &>/dev/null; then
  HAS_CLAUDE=true
  success "Claude Code found"
else
  warn "Claude Code not found — skill installation will be skipped."
  info "Install Claude Code later, then re-run this script to add the skill."
fi

# ── Step 2: Install or upgrade slackbuzz ─────────────────────────────
step "2/6: Install or upgrade slackbuzz"

if command -v slackbuzz &>/dev/null; then
  CURRENT_VERSION=$(slackbuzz version 2>/dev/null | head -1 || echo "unknown")
  info "slackbuzz is already installed ($CURRENT_VERSION)"
  info "Checking for updates..."
  if brew upgrade triptechtravel/tap/slackbuzz 2>/dev/null; then
    success "Upgraded to latest version"
  else
    success "Already on the latest version"
  fi
else
  info "Installing slackbuzz via Homebrew..."
  brew install triptechtravel/tap/slackbuzz
  success "Installed slackbuzz"
fi

INSTALLED_VERSION=$(slackbuzz version 2>/dev/null | head -1 || echo "unknown")
info "Version: $INSTALLED_VERSION"

# ── Step 3: Install Claude Code skill ────────────────────────────────
step "3/6: Install Claude Code skill"

if [ "$HAS_CLAUDE" = false ]; then
  info "Skipping — Claude Code is not installed."
else
  SKILL_DIR="$HOME/.claude/skills/slackbuzz-cli"

  if [ -L "$SKILL_DIR" ]; then
    # Existing symlink (from `make install-skill` in a local clone)
    TARGET=$(readlink "$SKILL_DIR")
    success "Skill already linked (symlink -> $TARGET)"
    info "To update, pull the repo and the symlink picks up changes automatically."
  elif [ -f "$SKILL_DIR/SKILL.md" ]; then
    success "Skill already installed"
    info "Updating to latest version..."
    SKILL_URL="https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/skills/slackbuzz-cli/SKILL.md"
    curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
    success "Skill updated"
  else
    info "Downloading Claude Code skill..."
    mkdir -p "$SKILL_DIR"
    SKILL_URL="https://raw.githubusercontent.com/triptechtravel/slackbuzz-cli/main/skills/slackbuzz-cli/SKILL.md"
    curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
    success "Skill installed to $SKILL_DIR"
  fi

  info "Claude Code will now use slackbuzz for all Slack operations."
fi

# ── Step 4: Authenticate ────────────────────────────────────────────
step "4/6: Authenticate with Slack"

NEED_BOT=true
NEED_USER=true

if command -v slackbuzz &>/dev/null; then
  AUTH_STATUS=$(slackbuzz auth status 2>&1 || true)
  if echo "$AUTH_STATUS" | grep -qi "bot.*authenticated\|bot.*valid\|bot token.*ok\|bot token.*set"; then
    NEED_BOT=false
  fi
  if echo "$AUTH_STATUS" | grep -qi "user.*authenticated\|user.*valid\|user token.*ok\|user token.*set"; then
    NEED_USER=false
  fi
fi

if [ "$NEED_BOT" = false ] && [ "$NEED_USER" = false ]; then
  success "Both tokens are already configured"
  slackbuzz auth status 2>&1 | sed 's/^/  /'
else
  info "slackbuzz needs two Slack tokens. Your team lead can share these."
  printf "\n"
  info "${BOLD}Bot token${RESET} (xoxb-...)  — shared across the team"
  info "  Used for: reading channels, listing users, reactions"
  info "${BOLD}User token${RESET} (xoxp-...) — unique per person"
  info "  Used for: sending messages as you, search, DMs, status"
  printf "\n"
  info "Get both from your Slack app's OAuth page:"
  info "  https://api.slack.com/apps → Your App → OAuth & Permissions"
  printf "\n"

  if [ "$NEED_BOT" = true ]; then
    info "Paste your bot token (xoxb-...) when prompted:"
    slackbuzz auth login --bot-token
    success "Bot token saved"
  else
    success "Bot token already configured"
  fi

  if [ "$NEED_USER" = true ]; then
    info "Paste your user token (xoxp-...) when prompted:"
    slackbuzz auth login --user-token
    success "User token saved"
  else
    success "User token already configured"
  fi
fi

# ── Step 5: Health check ────────────────────────────────────────────
step "5/6: Health check"

info "Running slackbuzz doctor..."
printf "\n"
if slackbuzz doctor; then
  printf "\n"
  success "All checks passed"
else
  printf "\n"
  warn "Some checks failed — see output above."
  info "Common fixes: re-run 'slackbuzz auth login' or ask your team lead to add missing scopes."
fi

# ── Step 6: Summary ─────────────────────────────────────────────────
step "6/6: You're all set"

printf "\n"
printf "  %s┌──────────────────────────────────────────────────┐%s\n" "$CYAN" "$RESET"
printf "  %s│%s  ${BOLD}slackbuzz quick reference${RESET}                        %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s├──────────────────────────────────────────────────┤%s\n" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s  ${BOLD}Inbox & activity${RESET}                                %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz activity           # mentions       %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz inbox               # alias         %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s  ${BOLD}Messaging${RESET}                                       %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz send '#general' \"Hi\"  # send msg   %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz send @alice \"Hey\"     # send DM    %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz message search \"query\"             %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s  ${BOLD}Status${RESET}                                          %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz status               # view        %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz status set \"Busy\" :headphones:     %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s  ${BOLD}Diagnostics${RESET}                                     %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz doctor               # check auth  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    slackbuzz auth status           # token info  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"

if [ "$HAS_CLAUDE" = true ]; then
printf "  %s│%s  ${BOLD}Claude Code (natural language)${RESET}                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    \"Check my Slack inbox\"                       %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    \"Send @alice a message about the PR\"         %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s    \"Search Slack for deploy instructions\"       %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s│%s                                                  %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
fi

printf "  %s│%s  Docs: triptechtravel.github.io/slackbuzz-cli   %s│%s\n" "$CYAN" "$RESET" "$CYAN" "$RESET"
printf "  %s└──────────────────────────────────────────────────┘%s\n" "$CYAN" "$RESET"
printf "\n"
