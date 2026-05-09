---
title: Installation
description: Install the slackbuzz CLI via Go, Homebrew, or binary release.
---

There are three ways to install the `slackbuzz` CLI.

## Go

If you have Go 1.25 or later installed, use `go install`:

```sh
go install github.com/triptechtravel/slackbuzz-cli/cmd/slackbuzz@latest
```

This places the binary in your `$GOBIN` directory (typically `$HOME/go/bin`). Make sure that directory is in your `PATH`.

## Homebrew

On macOS and Linux, install via Homebrew:

```sh
brew install triptechtravel/tap/slackbuzz
```

To upgrade later:

```sh
brew upgrade triptechtravel/tap/slackbuzz
```

## Binary releases

Download a prebuilt binary for your platform from the [GitHub releases page](https://github.com/triptechtravel/slackbuzz-cli/releases).

1. Download the archive for your OS and architecture.
2. Extract it.
3. Move the `slackbuzz` binary to a directory in your `PATH` (for example, `/usr/local/bin`).

```sh
# Example for macOS arm64
tar xzf slackbuzz_darwin_arm64.tar.gz
sudo mv slackbuzz /usr/local/bin/
```

## Verify the installation

After installing, confirm the CLI is available:

```sh
slackbuzz version
```

This prints the version, commit SHA, and build date.

## Shell completions

Generate completion scripts for your shell to enable tab completion of commands and flags.

```sh
# Bash
source <(slackbuzz completion bash)

# Zsh
source <(slackbuzz completion zsh)
# Or install permanently:
slackbuzz completion zsh > "${fpath[1]}/_slackbuzz"

# Fish
slackbuzz completion fish | source

# PowerShell
slackbuzz completion powershell | Out-String | Invoke-Expression
```

## Dependencies

The `digest` command optionally uses the [ClickUp CLI](https://github.com/triptechtravel/clickup-cli) and [GitHub CLI](https://cli.github.com/) (`gh`) for cross-tool briefings. Sections are gracefully skipped if these CLIs are not installed.
