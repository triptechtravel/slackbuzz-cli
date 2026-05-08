---
title: Contributing
description: Development setup, project structure, and contribution guidelines.
---

# Contributing

Thank you for your interest in contributing to slackbuzz-cli! This page covers development setup, project structure, and contribution guidelines.

## Development setup

```sh
# Clone the repository
git clone https://github.com/triptechtravel/slackbuzz-cli.git
cd slackbuzz-cli

# Build
go build -o slackbuzz ./cmd/slackbuzz

# Run tests
go test ./...

# Run vet
go vet ./...

# Install locally
go install ./cmd/slackbuzz

# Regenerate CLI reference docs (Starlight markdown)
make docs

# Regenerate the typed Slack client from api/specs/slack_web.json
make api-gen

# Regenerate pkg/cmd/app/manifest.go from method usage
make manifest-gen

# CI gate — fail if generated artifacts have drifted
make verify-gen
```

Requires Go 1.25 or later.

For the full design — codegen pipeline, spec patches, transport layer — see the [Architecture](./architecture/) page.

## Project structure

```
cmd/slackbuzz/        Entry point
cmd/gen-docs/         Auto-generates CLI reference markdown (make docs)
internal/             Internal packages (not importable)
  api/                HTTP client, rate limiting, error handling
  app/                Bootstrap and root command wiring
  auth/               Keyring storage, token validation
  build/              Version info (injected via ldflags)
  config/             YAML config loading
  iostreams/          TTY detection, color support, stream abstraction
  tableprinter/       TTY-aware table rendering
  prompter/           Interactive prompts (survey-based)
  browser/            Cross-platform browser opening
  text/               Emoji rendering, Slack text formatting, deeplinks
pkg/
  cmdutil/            Dependency injection, JSON flags, auth middleware
  cmd/                Command implementations
    activity/         Inbox: mentions, DMs, threads (alias: inbox)
    app/              Slack app creation with manifest
    auth/             login, logout, status
    channel/          list, info
    completion/       Shell completions
    digest/           Cross-tool morning briefing
    dm/               DM conversation listing
    later/            Saved/bookmarked items
    message/          list, send, search
    notify/           Block Kit notifications
    react/            Emoji reactions
    status/           Slack status set/clear
    thread/           Thread replies and links
    threads/          Thread listing
    user/             list, info
    version/          Version
```

## Running tests

```sh
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Specific package
go test ./pkg/cmd/activity/...

# Specific test
go test -run TestEnrichLinks ./pkg/cmd/activity/
```

## Code style

- Follow standard Go formatting (`gofmt`)
- Wrap errors with context: `fmt.Errorf("failed to fetch messages: %w", err)`
- Use table-driven tests where appropriate
- Prefer the standard library unless a dependency adds clear value

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `go test ./...` and `go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`
3. GitHub Actions builds binaries for all platforms
4. Updates the Homebrew formula in `triptechtravel/homebrew-tap`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/triptechtravel/slackbuzz-cli/blob/main/LICENSE).
