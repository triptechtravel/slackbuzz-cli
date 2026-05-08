---
title: Architecture
description: How slackbuzz is built — codegen pipeline, package layout, scope handling.
---

# Architecture

This page is for contributors. Users don't need to read it.

## Layout

```
api/specs/slack_web.json     # Slack's published OpenAPI 2.0 spec (committed)
cmd/
  gen-api/                   # spec → typed Slack client
    main.go                    # type + operation + scope generators
    patches.go                 # spec-quirk fixups
  gen-manifest/              # source → app manifest
    main.go                    # AST-walks pkg/cmd for slackapi.* calls
  gen-docs/                  # cobra command tree → markdown reference pages
internal/
  slackapi/                  # the typed Slack Web API client
    client.go                  # hand-written transport (Do, BaseResponse, APIError)
    blocks.go                  # block-kit composition helpers
    files.go                   # multipart files.upload (legacy endpoint)
    operations.go              # hand-augmented ops not in spec (search.files)
    types.go                   # hand-augmented types
    types.gen.go               # generated: 19 shared structs + 174 method param/response types
    operations.gen.go          # generated: one wrapper per Slack method
    scopes.gen.go              # generated: method → required scopes map
  api/                       # transport-level concerns wrapping slackapi
    client.go                  # rate-limited http.Client + slackapi.Client
    fuzzy.go                   # tiered exact / contains / fuzzy resolver helper
    resolve.go                 # @user, #channel → ID (uses fuzzy)
    rate_limit.go              # 429 / Retry-After tracking
  recent/                    # last-used targets per command type
  auth/                      # token storage + validation
pkg/
  cmd/                       # command implementations (cobra)
    app/                       # `app create`, `app update` — manifest is generated
      manifest.go                # GENERATED — do not edit
    …
  cmdutil/                   # shared command utilities
    bulk_args.go               # repeated positional arg helpers
    json_flags.go              # --json / --jq / --template
```

## The codegen pipeline

Three generators, three sources of truth:

```
                        ┌─────────────────────────────┐
api/specs/slack_web.json│    cmd/gen-api              │
                        │  reads spec  →  emits…      │──┐
                        └─────────────────────────────┘  │
                                                         ▼
                        ┌─────────────────────────────────────────────────┐
                        │ internal/slackapi/                              │
                        │   types.gen.go      (174 method param/response  │
                        │                       structs + 19 shared types)│
                        │   operations.gen.go (174 wrapper functions)     │
                        │   scopes.gen.go     (method → []scope map)      │
                        └────────────────────────┬────────────────────────┘
                                                 │
                                                 ▼
pkg/cmd/**/*.go calls    ┌─────────────────────────────┐
slackapi.<Method>()      │   cmd/gen-manifest          │
                         │  AST-walks for slackapi.*   │
                         │  calls, looks up each method's│
                         │  scopes, emits manifest      │
                         └────────────────────────────┬─┘
                                                      ▼
                                  pkg/cmd/app/manifest.go (GENERATED)
                                  consumed by `slackbuzz app create / app update`
```

### Why this matters

Adding a `slackapi.SomeNewMethod(...)` call somewhere in `pkg/cmd/` triggers a manifest update on the next `make manifest-gen`. CI runs `make verify-gen` and fails the PR if the manifest drifts from method usage. This makes scope-drift bugs (the kind that broke DM reads in early development) structurally impossible to ship.

## The spec patches

Slack's OpenAPI spec has a handful of consistent quirks that hurt generated-code quality:

- **Inline-vs-ref drift** — `conversations.history` uses `$ref: objs_message`, but `conversations.replies` inlines the same shape. Patches restore the missing `$ref`.
- **Empty `objs_conversation`** — has no `type` and no `properties`, so the generator can't emit a struct. Patches redirect refs to `objs_channel` (the populated definition).
- **Empty `objs_user`** — same problem, no fields in the spec. Patches inject a synthetic `objs_user` with the fields commands actually consume (id, name, deleted, profile ref).
- **Inline topic/purpose objects** — `objs_channel.topic` and `.purpose` are `{value, creator, last_set}` inline. Patches synthesise `objs_topic_purpose` and rewrite the refs so callers get `*TopicPurpose` instead of `map[string]any`.
- **Nullable scalars as tuple-items** — `{"items":[{$ref}, {type:"null"}]}` instead of OpenAPI 3's `nullable: true`. The generator's `goTypeForSchema` peels these back to the underlying primitive.
- **Timestamp number-vs-string** — `ts`, `latest`, `oldest`, `thread_ts` are typed as `number` but sub-second precision matters; treated as `string` everywhere.

Each patch lives in `cmd/gen-api/patches.go` with a comment explaining what it changes and why. Patches are organised into:

- **`injectSyntheticDefinitions`** — adds new `objs_*` entries to the spec for shapes Slack uses but doesn't name.
- **`patchInlineMessageArrays`** — generic detection of inline message-shaped arrays.
- **`patchKnownInlineRefs`** — surgical path-keyed rewrites (e.g. `conversations.replies → messages → objs_message`).
- **`patchDefinitionFieldRefs`** — rewrites a property inside a definition to ref a sibling definition.

Add new entries when you find another quirk.

## The transport layer

`internal/slackapi/client.go` is the hand-written transport. Every generated wrapper bottoms out in `Do()`:

```go
func Do(ctx context.Context, c *Client, method string, form url.Values, out Envelope) ([]byte, error)
```

It:

1. POSTs `https://slack.com/api/<method>` with the bearer token + form body
2. Reads the response and JSON-decodes into `out` (which embeds `BaseResponse`)
3. If `ok=false`, builds a typed `*APIError` with `Code`, `Method`, and (for `missing_scope`) `NeededScopes`/`ProvidedScopes`
4. Returns the raw body as `[]byte` so callers can re-decode for fields the spec doesn't model

Rate limiting and retry on 429 happen one layer up, in `internal/api/http.go:NewHTTPClient`, which wraps the transport before slackapi sees it.

## Caller error matching

Use `errors.Is` against the sentinels:

```go
import "github.com/triptechtravel/slackbuzz-cli/internal/slackapi"

resp, err := slackapi.ChatPostMessage(ctx, client, params)
if errors.Is(err, slackapi.ErrChannelNotFound) {
    // Handle stale channel ID
}
if errors.Is(err, slackapi.ErrMissingScope) {
    var apiErr *slackapi.APIError
    errors.As(err, &apiErr)
    fmt.Printf("Need: %v\n", apiErr.NeededScopes)
}
```

The full sentinel list lives in `internal/slackapi/client.go` — `ErrMissingScope`, `ErrChannelNotFound`, `ErrUserNotFound`, `ErrNotInChannel`, `ErrIsArchived`, `ErrMessageNotFound`, etc.

## Make targets

```sh
make api-spec       # download Slack's OpenAPI spec
make api-gen        # regenerate internal/slackapi/*.gen.go from the spec
make manifest-gen   # regenerate pkg/cmd/app/manifest.go from method usage
make verify-gen     # CI gate — fail if generated artifacts have drifted
make docs           # regenerate command-reference markdown
make test           # run all tests
make smoke          # exercise the typed client against a real Slack workspace
```

## `go generate` markers

`internal/slackapi/types.go` and `pkg/cmd/app/app.go` carry `//go:generate` directives so contributors can run `go generate ./...` instead of memorising Make targets:

```go
//go:generate go run ../../cmd/gen-api -spec ../../api/specs/slack_web.json -out-dir .
```

The Make targets remain as the orchestration entry-points (CI uses them); `go generate` is additional, not replacing.

## CI

`.github/workflows/ci.yml` runs four jobs in parallel on every push and PR:

- **test** — `go test -race ./...` and `go vet ./...`
- **build** — confirms the binary compiles
- **verify-gen** — `make verify-gen` fails if generated artifacts have drifted from source. *This is the structural guarantee that the manifest never ships out-of-sync with method usage.*
- **lint** — gofmt + golangci-lint (staticcheck, gosimple, etc.)

## Adding a Slack method to slackbuzz

If the method is in Slack's spec (look in `api/specs/slack_web.json`):

1. `make api-gen` — already there, you don't need to do anything else.
2. Call `slackapi.<MethodName>(ctx, client, params)` from your command.
3. `make manifest-gen` — auto-adds required scopes to the manifest.
4. `make verify-gen` — confirms there's no drift.

If the method is NOT in Slack's spec (rare — search.files is the current example):

1. Add a `Params` and `Response` type to `internal/slackapi/types.go`.
2. Add a wrapper function to `internal/slackapi/operations.go`.
3. Update `cmd/gen-manifest/main.go`'s `handAugmented` map so the manifest knows the API name.
4. Use it the same way as generated methods.
