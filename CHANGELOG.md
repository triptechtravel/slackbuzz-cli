# Changelog

All notable changes to slackbuzz-cli are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.12.0]

### Fixed

- **`message edit` skipped the outgoing-text pipeline** — edited messages
  were sent to `chat.update` verbatim, so Markdown arrived unconverted
  (`**bold**` rendered literally), `@mentions` didn't ping, and shell
  escape artifacts weren't stripped. Edits now go through the same
  pipeline as `message send`.
- **`notify --message` skipped Markdown→mrkdwn conversion and shell
  unescape** (it did resolve mentions). Custom notification messages render
  via mrkdwn blocks, so raw Markdown arrived broken. Now normalized like
  `send`/`edit`.

### Added

- `message edit --raw` to disable Markdown→mrkdwn auto-conversion,
  matching `message send --raw`.

### Internal

- Outgoing-text normalization (shell unescape + Markdown→mrkdwn + format
  hints) consolidated into `internal/text.NormalizeOutgoing` and
  `cmdutil.PrintFormatHints`, used by `send`, `edit`, and `notify` so the
  pipeline can't drift between commands again. `unescapeShellArtifacts`
  moved from `pkg/cmd/message` to `internal/text.UnescapeShellArtifacts`.

## [0.11.1]

### Fixed

- **`slackbuzz file upload` was broken in v0.11.0** — Slack hard-deprecated
  the legacy `files.upload` endpoint and started returning `method_deprecated`
  on every fresh request. Replaced the single-call `files.upload` wrapper
  with the modern 3-step external-upload flow:
  `files.getUploadURLExternal` → presigned multipart POST →
  `files.completeUploadExternal`. The public `slackapi.UploadFile` signature
  is unchanged, so callers don't need to touch anything. Added unit tests
  (mocked 3-step flow) and verified end-to-end against live Slack.
- **Manifest reflects the two new endpoints.** `gen-manifest`'s
  `handAugmented` map now supports a single Go function mapping to multiple
  API methods so scopes for both upload endpoints are picked up.

## [0.11.0]

### Added

- **`slackbuzz app update`** — push the latest scope manifest to an existing
  Slack app and re-authenticate in one flow. Use after a release that adds
  new scope requirements.
- **Spec-driven typed Slack client** — generated from Slack's published
  OpenAPI 2.0 spec (`api/specs/slack_web.json`). 174 methods × typed
  params + responses. New `cmd/gen-api` tool walks the spec and emits
  `internal/slackapi/{types,operations,scopes}.gen.go`.
- **Auto-derived app manifest** — `cmd/gen-manifest` AST-walks
  `pkg/cmd/` for `slackapi.<Method>(...)` calls and writes
  `pkg/cmd/app/manifest.go` from the corresponding scope union. CI's
  `make verify-gen` fails on drift.
- **Fuzzy resolution** — three-tier (exact / contains / fuzzy) channel
  and user resolution with "did you mean: …?" suggestions on miss.
- **Recent-context defaults** — `slackbuzz dm` (no args) opens recent
  DMs; `message list/send`, `notify` record their target into
  `~/.config/slack/recent.json` for future no-arg defaults.
- **Typed sentinel errors** — `slackapi.ErrMissingScope`,
  `ErrChannelNotFound`, `ErrRatelimited`, etc. Match with `errors.Is`
  instead of string-matching error text.
- **Bulk argument helpers** — `pkg/cmdutil/bulk_args.go` for commands
  taking repeated positional args (whitespace-split, stdin readers).
- **Live-Slack smoke test** — `make smoke` exercises the typed client
  against a real workspace. Not wired to PR CI (requires real tokens).
- **CI gates** — `make verify-gen` and `golangci-lint` jobs added to
  `.github/workflows/ci.yml`. Tests now run with `-race`.
- **Architecture documentation** — new docs page describing the codegen
  pipeline, spec patches, transport, error matching, and Make targets.

### Changed

- **Removed `slack-go/slack` dependency.** Every Slack API call now goes
  through `internal/slackapi`. The `slack-go` import is gone from `go.mod`.
- **Manifest is now generated.** `pkg/cmd/app/manifest.go` no longer
  hand-maintained; runs `make manifest-gen` to update from method usage.
- **Error helpers use sentinels.** `IsMissingScopeError`, `IsAuthRelatedSlackError`
  rewritten to use `errors.Is` against typed sentinels rather than
  string-matching the formatted error text.

### Fixed

- **`channel_not_found` / `missing_scope` text-matching footgun** — error
  helpers no longer break if `APIError.Error()` formatting changes.
- **Generated `Type_` field-name suffix** — was applied because of a
  bogus keyword check; `Type` / `Range` / `Map` etc. aren't Go keywords
  post-`camelInitUpper`. Fields now use the natural Go name.
- **`recent.Load()` exposed mutable cache** — `Load()` and `Save()` are
  now unexported; external callers go through `Push`/`Last`/`List` which
  return defensive copies.
- **Typed responses for `users.list`/`users.info`/`channel info`** —
  `objs_user` is empty in Slack's spec, and `objs_channel.{topic,purpose}`
  are inlined; spec patches in `cmd/gen-api/patches.go` synthesise the
  missing types so callers get `*User` and `*TopicPurpose` instead of
  `map[string]any` indexing.

### Internal

- New top-level packages: `internal/slackapi`, `internal/recent`.
- New CLIs: `cmd/gen-api`, `cmd/gen-manifest`.
- Spec-quirk patches consolidated in `cmd/gen-api/patches.go` with
  documented entries per quirk.
- `//go:generate` markers added to `internal/slackapi/types.go` and
  `pkg/cmd/app/app.go` so `go generate ./...` refreshes the pipeline.

## [0.10.4] — earlier

See git history for releases prior to this changelog.
