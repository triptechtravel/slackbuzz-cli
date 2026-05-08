BINARY := slackbuzz
MODULE := github.com/triptechtravel/slackbuzz-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X $(MODULE)/internal/build.Version=$(VERSION) \
	-X $(MODULE)/internal/build.Commit=$(COMMIT) \
	-X $(MODULE)/internal/build.Date=$(DATE)

.PHONY: build install test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/slackbuzz

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/slackbuzz

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/

.PHONY: docs
docs:
	go run ./cmd/gen-docs

.PHONY: snapshot
snapshot:
	goreleaser --snapshot --clean

.PHONY: install-skill
install-skill:
	@mkdir -p $(HOME)/.claude/skills
	@ln -sfn $(CURDIR)/skills/slackbuzz-cli $(HOME)/.claude/skills/slackbuzz-cli
	@echo "Linked Claude Code skill: slackbuzz-cli → ~/.claude/skills/slackbuzz-cli"

.PHONY: setup
setup:
	@bash scripts/setup.sh

# ── API spec + codegen ────────────────────────────────────────────
# Single source of truth for Slack methods, parameters, scopes, and types.
# Mirrors the clickup-cli pattern in spirit: spec lives in `api/specs/`,
# codegen tools live in `cmd/gen-*`, generated output is committed.
SLACK_SPEC_URL := https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json

api/specs/slack_web.json:
	@mkdir -p api/specs
	curl -sfL -o $@.tmp $(SLACK_SPEC_URL)
	# Slack's published spec embeds an example xoxb- token in oauth.v2.access's
	# response example, which trips GitHub Push Protection. Redact it on download
	# so the committed file is always safe.
	@sed -E 's/"xoxb-[A-Za-z0-9-]+"/"xoxb-EXAMPLE-BOT-TOKEN-REDACTED"/g' $@.tmp > $@
	@rm -f $@.tmp
	@echo "Downloaded Slack web spec ($$(wc -c < $@ | tr -d ' ') bytes, $$(jq '.paths | length' $@) methods)"

.PHONY: api-spec
api-spec: api/specs/slack_web.json

.PHONY: api-spec-refresh
api-spec-refresh:
	@rm -f api/specs/slack_web.json
	@$(MAKE) api/specs/slack_web.json

# Regenerate the canonical app-manifest from the union of scopes for every
# Slack method actually called by the CLI. Eliminates the manual-drift class
# of bugs (where the manifest is stale relative to what commands need).
.PHONY: manifest-gen
manifest-gen: api-spec
	@echo "Regenerating pkg/cmd/app/manifest.go from spec + command method usage..."
	go run ./cmd/gen-manifest \
		-spec api/specs/slack_web.json \
		-cmd-root pkg/cmd \
		-out pkg/cmd/app/manifest.go
	gofmt -w pkg/cmd/app/manifest.go

# Regenerate the typed Slack client (params, response types, ops, scopes)
# from the spec. The single gen-api tool emits everything into
# internal/slackapi/{types,operations,scopes}.gen.go.
.PHONY: api-gen
api-gen: api-spec
	@echo "Generating Slack typed client (types + operations + scopes)..."
	go run ./cmd/gen-api -spec api/specs/slack_web.json -out-dir internal/slackapi
	gofmt -w internal/slackapi/

# CI hygiene: fail if generated artifacts drift from committed.
.PHONY: verify-gen
verify-gen: manifest-gen api-gen
	@if ! git diff --quiet pkg/cmd/app/manifest.go internal/slackapi/; then \
		echo "ERROR: generated files are out of date. Run 'make manifest-gen api-gen' and commit."; \
		git diff --stat pkg/cmd/app/manifest.go internal/slackapi/; \
		exit 1; \
	fi

# Live smoke test against a real Slack workspace. Uses the currently
# logged-in tokens; never run in PR CI. Override BIN to test a local
# build: `BIN=./bin/slackbuzz make smoke`.
.PHONY: smoke
smoke:
	@./scripts/smoke.sh
