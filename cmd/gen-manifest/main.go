// Command gen-manifest derives the Slack-app manifest scope list from
// the union of scopes for every Slack API method actually called by the
// CLI, then regenerates pkg/cmd/app/manifest.go.
//
// Eliminates the manual-drift class of bugs (where the manifest lists a
// stale scope set and commands fail with `missing_scope` until someone
// notices). After running `make api-gen` to refresh
// internal/slackapi/scopes.gen.go, run `make manifest-gen` and the
// manifest follows.
//
// Usage:
//
//	go run ./cmd/gen-manifest \
//	    -spec api/specs/slack_web.json \
//	    -cmd-root pkg/cmd \
//	    -out pkg/cmd/app/manifest.go
//
// Mechanism:
//
//  1. Read the spec — pull every method's required scopes from `security`
//     blocks (same as cmd/gen-api).
//  2. AST-walk the command tree, looking for `slackapi.<GoName>(…)` calls.
//     Map the Go name back to its dotted API name and accumulate a set
//     of called methods.
//  3. Compute the union of scopes across all called methods.
//  4. Split into bot vs user scope lists, filtering Slack's `:bot`/`:user`
//     qualifier suffixes appropriately.
//  5. Render pkg/cmd/app/manifest.go from a template.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ─── spec loading (minimal subset; enough for scope extraction) ────────────

type spec struct {
	Paths map[string]map[string]operation `json:"paths"`
}

type operation struct {
	Security []map[string][]string `json:"security,omitempty"`
}

func loadSpec(path string) (map[string][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for p, verbs := range s.Paths {
		apiName := strings.TrimPrefix(p, "/")
		for _, op := range verbs {
			seen := map[string]bool{}
			var scopes []string
			for _, sec := range op.Security {
				for _, ss := range sec {
					for _, sc := range ss {
						if seen[sc] {
							continue
						}
						seen[sc] = true
						scopes = append(scopes, sc)
					}
				}
			}
			sort.Strings(scopes)
			out[apiName] = scopes
		}
	}
	return out, nil
}

// ─── name munging (mirrors cmd/gen-api) ────────────────────────────────────

var commonAcronym = map[string]bool{
	"ID": true, "URL": true, "API": true, "TS": true, "UID": true,
	"DM": true, "IP": true, "JSON": true, "CSV": true, "MPIM": true,
}

func camelInitUpper(s string) string {
	if s == "" {
		return s
	}
	upper := strings.ToUpper(s)
	if commonAcronym[upper] {
		return upper
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}

func apiNameToGoName(apiName string) string {
	parts := strings.Split(apiName, ".")
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(camelInitUpper(p))
	}
	return b.String()
}

// ─── AST walking ───────────────────────────────────────────────────────────

// findCalledMethods walks the Go source tree under cmdRoot and returns the
// set of Slack API method names called via `slackapi.<GoName>(...)`.
//
// It also recognises the hand-augmented operations (SearchFiles → search.files,
// UploadFile → files.getUploadURLExternal + files.completeUploadExternal) so
// the manifest stays correct even when the generated operation set isn't the
// only source of truth. UploadFile maps to TWO methods because Slack's modern
// upload flow is a 3-step external dance.
func findCalledMethods(cmdRoot string, goToAPI map[string]string) (map[string]bool, error) {
	called := map[string]bool{}

	// Hand-augmented mappings — keep in sync with internal/slackapi/operations.go
	// and internal/slackapi/files.go. A single Go function can map to multiple
	// underlying API methods (e.g. UploadFile uses two endpoints).
	handAugmented := map[string][]string{
		"SearchFiles": {"search.files"},
		"UploadFile":  {"files.getUploadURLExternal", "files.completeUploadExternal"},
	}

	fset := token.NewFileSet()
	err := filepath.WalkDir(cmdRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Don't introspect manifest.go itself — it's the output target.
		if strings.HasSuffix(path, "/app/manifest.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Name != "slackapi" {
				return true
			}
			funcName := sel.Sel.Name
			if apis, ok := handAugmented[funcName]; ok {
				for _, api := range apis {
					called[api] = true
				}
				return true
			}
			if api, ok := goToAPI[funcName]; ok {
				called[api] = true
			}
			return true
		})
		return nil
	})
	return called, err
}

// ─── scope union + bot/user split ──────────────────────────────────────────

// classifyScopes splits a flat set of scopes into bot-side and user-side
// lists, filtering Slack's :bot / :user qualifier suffixes. Scopes without
// a qualifier go to both.
func classifyScopes(scopes []string) (bot, user []string) {
	botSet := map[string]bool{}
	userSet := map[string]bool{}
	for _, s := range scopes {
		switch {
		case strings.HasSuffix(s, ":bot"):
			// chat:write:bot → chat:write on the bot side; user side ignores.
			botSet[strings.TrimSuffix(s, ":bot")] = true
		case strings.HasSuffix(s, ":user"):
			userSet[strings.TrimSuffix(s, ":user")] = true
		default:
			botSet[s] = true
			userSet[s] = true
		}
	}
	for s := range botSet {
		bot = append(bot, s)
	}
	for s := range userSet {
		user = append(user, s)
	}
	sort.Strings(bot)
	sort.Strings(user)
	return bot, user
}

// baseline scopes the CLI always wants regardless of which methods land in
// the source tree. These give the bot a usable presence on install and
// support the install-time auth.test handshake.
var baselineBotScopes = []string{
	"users:read", // resolver needs users.list
	"emoji:read", // emoji rendering
	"chat:write", // posting
	"channels:read",
	"groups:read",
	"im:read",
	"mpim:read",
	"channels:history",
	"groups:history",
	"im:history",
	"mpim:history",
	"reactions:read",
	"reactions:write",
	"files:read",
	"files:write",
}
var baselineUserScopes = []string{
	"users:read",
	"users.profile:read",
	"users.profile:write",
	"chat:write",
	"channels:read",
	"groups:read",
	"im:read",
	"mpim:read",
	"channels:history",
	"groups:history",
	"im:history",
	"mpim:history",
	"reactions:write",
	"files:read",
	"files:write",
	"search:read",
	"stars:read",
	"stars:write",
	"im:write",
}

func mergeBaseline(generated, baseline []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(generated)+len(baseline))
	for _, s := range generated {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, s := range baseline {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ─── output ────────────────────────────────────────────────────────────────

type templateData struct {
	BotScopes  []string
	UserScopes []string
	Methods    []string
}

var manifestTmpl = template.Must(template.New("manifest").Parse(`// Code generated by cmd/gen-manifest. DO NOT EDIT.
//
// Regenerate with ` + "`make manifest-gen`" + `. The scope lists below are
// the union of:
//
//   1. Scopes required by every Slack API method called from pkg/cmd/...
//      (auto-derived via AST walk + spec lookup)
//   2. A baseline set of scopes the CLI wants regardless (resolver, auth
//      handshake, etc.) — see cmd/gen-manifest/main.go:baseline*Scopes
//
// Adding a slackapi.<Method>(...) call somewhere in pkg/cmd/ triggers an
// automatic scope update on the next ` + "`make manifest-gen`" + `; CI's
// ` + "`make verify-gen`" + ` will fail if the manifest drifts.

package app

// appManifest is the canonical Slack app manifest used by ` + "`app create`" + `
// and ` + "`app update`" + `. The two consumer commands compose it as a JSON
// payload to apps.manifest.{create,update}.
//
// Methods this scope set covers (alphabetised):
{{range .Methods}}//   - {{.}}
{{end}}
var appManifest = map[string]interface{}{
	"display_information": map[string]interface{}{
		"name":             "SlackBuzz CLI",
		"description":      "CLI tool for Slack messaging, channel management, and search",
		"background_color": "#1a1a2e",
	},
	"features": map[string]interface{}{
		"bot_user": map[string]interface{}{
			"display_name":  "SlackBuzz",
			"always_online": false,
		},
	},
	"oauth_config": map[string]interface{}{
		"scopes": map[string]interface{}{
			"bot": []string{
{{range .BotScopes}}				{{printf "%q" .}},
{{end}}			},
			"user": []string{
{{range .UserScopes}}				{{printf "%q" .}},
{{end}}			},
		},
	},
	"settings": map[string]interface{}{
		"org_deploy_enabled":     false,
		"socket_mode_enabled":    false,
		"token_rotation_enabled": false,
	},
}
`))

// ─── main ──────────────────────────────────────────────────────────────────

func main() {
	specPath := flag.String("spec", "api/specs/slack_web.json", "Path to Slack OpenAPI spec")
	cmdRoot := flag.String("cmd-root", "pkg/cmd", "Source root to scan for slackapi.* calls")
	outPath := flag.String("out", "pkg/cmd/app/manifest.go", "Output path for the generated manifest")
	flag.Parse()

	methodScopes, err := loadSpec(*specPath)
	if err != nil {
		log.Fatalf("load spec: %v", err)
	}

	// Build reverse map: GoName → APIName for the AST walk.
	goToAPI := map[string]string{}
	for api := range methodScopes {
		goToAPI[apiNameToGoName(api)] = api
	}

	called, err := findCalledMethods(*cmdRoot, goToAPI)
	if err != nil {
		log.Fatalf("walk source: %v", err)
	}
	if len(called) == 0 {
		log.Fatalf("no slackapi.<Method> calls found under %s — refusing to emit empty manifest", *cmdRoot)
	}

	// Collect scope union for called methods.
	scopeSet := map[string]bool{}
	for api := range called {
		for _, s := range methodScopes[api] {
			scopeSet[s] = true
		}
	}
	scopes := make([]string, 0, len(scopeSet))
	for s := range scopeSet {
		scopes = append(scopes, s)
	}
	bot, user := classifyScopes(scopes)
	bot = mergeBaseline(bot, baselineBotScopes)
	user = mergeBaseline(user, baselineUserScopes)

	methods := make([]string, 0, len(called))
	for m := range called {
		methods = append(methods, m)
	}
	sort.Strings(methods)

	f, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("create %s: %v", *outPath, err)
	}
	defer f.Close()
	if err := manifestTmpl.Execute(f, templateData{
		BotScopes:  bot,
		UserScopes: user,
		Methods:    methods,
	}); err != nil {
		log.Fatalf("render manifest: %v", err)
	}

	fmt.Printf("Generated manifest with %d methods, %d bot scopes, %d user scopes → %s\n",
		len(methods), len(bot), len(user), *outPath)
}
