// Command gen-api reads Slack's Web API OpenAPI 2.0 spec and generates
// typed Go wrappers for every operation, plus a method-to-scopes map,
// param structs, response structs (with typed fields), and shared object
// types from `definitions/*`.
//
// Generated files:
//
//	internal/slackapi/types.gen.go       — params, response structs, shared object types
//	internal/slackapi/operations.gen.go  — one function per Slack method
//	internal/slackapi/scopes.gen.go      — method → required-scopes map
//
// Usage:
//
//	go run ./cmd/gen-api -spec api/specs/slack_web.json
//
// Conventions:
//
//   - Method `auth.test` → Go function `AuthTest`, struct `AuthTestParams`,
//     `AuthTestResponse`. Multi-segment names (`admin.users.session.list`)
//     concatenate camel-cased segments.
//   - All Slack methods POST + form-encoded.
//   - The `token` query param is dropped — Authorization header carries it.
//   - Required vs optional: required params lose `omitempty`.
//   - Param types: string/integer/number/boolean/array(string) → Go natives.
//   - Response types: each method gets a typed struct with fields walked
//     from its OpenAPI response schema, embedding BaseResponse for ok/error
//     handling. A `Raw json.RawMessage` field is also emitted for power-user
//     access to fields the spec omits.
//   - Shared types: every `definitions/objs_*` becomes an exported Go struct.
//     `defs_*` (mostly primitive aliases like `defs_user_id` = string) are
//     skipped — the Go primitive is used directly at field sites.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ─── spec types ────────────────────────────────────────────────────────────

type spec struct {
	Paths       map[string]map[string]operation `json:"paths"`
	Definitions map[string]schema               `json:"definitions"`
}

type operation struct {
	OperationID string                `json:"operationId,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Parameters  []param               `json:"parameters,omitempty"`
	Responses   map[string]respWrap   `json:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type respWrap struct {
	Description string  `json:"description,omitempty"`
	Schema      *schema `json:"schema,omitempty"`
}

type param struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required,omitempty"`
	Type        string `json:"type,omitempty"`
	Items       *param `json:"items,omitempty"`
	Description string `json:"description,omitempty"`
}

// schema is the Swagger 2.0 / JSON Schema subset we model. Items and Type
// are kept as raw JSON because Slack's spec uses both with multiple shapes
// (Type: string OR array-of-strings for nullable; Items: object OR tuple).
// We collapse to the simplest interpretation rather than modelling all of it.
type schema struct {
	Type        json.RawMessage    `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Title       string             `json:"title,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Properties  map[string]*schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       json.RawMessage    `json:"items,omitempty"`
	Enum        []json.RawMessage  `json:"enum,omitempty"`
	OneOf       []schema           `json:"oneOf,omitempty"`
	AnyOf       []schema           `json:"anyOf,omitempty"`
	AllOf       []schema           `json:"allOf,omitempty"`
}

// typeStr returns the schema's primary type as a single string. If Type is
// an array (nullable types), the first non-"null" entry wins.
func (s *schema) typeStr() string {
	if len(s.Type) == 0 {
		return ""
	}
	// Single string: "type":"string"
	if s.Type[0] == '"' {
		var t string
		_ = json.Unmarshal(s.Type, &t)
		return t
	}
	// Array: "type":["integer","null"]
	if s.Type[0] == '[' {
		var ts []string
		if err := json.Unmarshal(s.Type, &ts); err != nil {
			return ""
		}
		for _, t := range ts {
			if t != "null" {
				return t
			}
		}
	}
	return ""
}

// itemsSchema parses Items as a single schema object, returning nil for the
// tuple-of-schemas case (handle that with itemsTuple).
func (s *schema) itemsSchema() *schema {
	if len(s.Items) == 0 {
		return nil
	}
	for _, b := range s.Items {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '[':
			return nil
		}
		break
	}
	var inner schema
	if err := json.Unmarshal(s.Items, &inner); err != nil {
		return nil
	}
	return &inner
}

// itemsTuple parses Items as a heterogeneous tuple. Slack uses this idiom
// for nullable scalars: items: [{$ref: defs_X}, {type: "null"}]. Returns
// nil when Items isn't tuple-shaped.
func (s *schema) itemsTuple() []schema {
	if len(s.Items) == 0 || s.Items[0] != '[' {
		return nil
	}
	var out []schema
	if err := json.Unmarshal(s.Items, &out); err != nil {
		return nil
	}
	return out
}

// firstNonNull returns the first tuple member that isn't a `{type:"null"}`
// sentinel. Used to peel Slack's nullable wrapping back to its underlying
// type.
func firstNonNull(tuple []schema) *schema {
	for i := range tuple {
		t := (&tuple[i]).typeStr()
		if t == "null" {
			continue
		}
		return &tuple[i]
	}
	return nil
}

// ─── intermediate model passed to templates ────────────────────────────────

type method struct {
	APIName           string
	GoName            string
	HTTPVerb          string
	Summary           string
	Params            []paramInfo
	HasNonTokenParams bool
	Scopes            []string
	RespFields        []respField // top-level fields from response.200 schema
}

type paramInfo struct {
	GoName    string
	APIName   string
	GoType    string
	Required  bool
	Comment   string
	URLEncode string
}

type respField struct {
	GoName   string
	APIName  string
	GoType   string
	Comment  string
	Required bool // required fields lose `omitempty` so callers can disambiguate "absent" from "zero-value"
}

type defType struct {
	GoName  string
	APIName string
	Fields  []respField // re-uses respField shape — same emission logic
	Title   string
}

// ─── main ──────────────────────────────────────────────────────────────────

var globalSpec *spec // referenced by ref-resolver helpers

func main() {
	specPath := flag.String("spec", "api/specs/slack_web.json", "Path to Slack OpenAPI 2.0 spec")
	outDir := flag.String("out-dir", "internal/slackapi", "Output directory")
	flag.Parse()

	raw, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("read spec: %v", err)
	}

	var s spec
	if err := json.Unmarshal(raw, &s); err != nil {
		log.Fatalf("parse spec: %v", err)
	}
	applyPatches(&s) // see patches.go
	globalSpec = &s

	defs := extractDefinitions(&s)
	methods := extractMethods(&s)
	if len(methods) == 0 {
		log.Fatalf("no methods extracted from spec")
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir out: %v", err)
	}

	if err := writeTypes(filepath.Join(*outDir, "types.gen.go"), methods, defs); err != nil {
		log.Fatalf("write types: %v", err)
	}
	if err := writeOperations(filepath.Join(*outDir, "operations.gen.go"), methods); err != nil {
		log.Fatalf("write operations: %v", err)
	}
	if err := writeScopes(filepath.Join(*outDir, "scopes.gen.go"), methods); err != nil {
		log.Fatalf("write scopes: %v", err)
	}

	fmt.Printf("Generated %d methods, %d shared types → %s/{types,operations,scopes}.gen.go\n",
		len(methods), len(defs), *outDir)
}

// ─── definition extraction ─────────────────────────────────────────────────

func extractDefinitions(s *spec) []defType {
	var out []defType
	for name, def := range s.Definitions {
		t := def.typeStr()
		// Skip primitive type aliases (defs_user_id, defs_ts, etc.) — they're
		// just patterned strings/numbers; the Go primitive is what we use.
		if strings.HasPrefix(name, "defs_") && t != "object" {
			continue
		}
		// Skip top-level arrays / unions; we only model object types as structs.
		if t != "" && t != "object" {
			continue
		}
		// Some `objs_*` are loosely defined with no properties — skip those too.
		if len(def.Properties) == 0 {
			continue
		}
		fields := schemaFields(&def)
		if len(fields) == 0 {
			continue
		}
		out = append(out, defType{
			GoName:  defNameToGoName(name),
			APIName: name,
			Title:   def.Title,
			Fields:  fields,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GoName < out[j].GoName })
	return out
}

// schemaFields walks an object schema's properties and emits respField entries.
//
// Properties listed in the schema's `required` array lose `omitempty` so
// JSON marshalling distinguishes "absent" from "zero-value". Mirrors the
// clickup-cli / oapi-codegen idiom.
func schemaFields(s *schema) []respField {
	requiredSet := map[string]bool{}
	for _, r := range s.Required {
		requiredSet[r] = true
	}
	var fields []respField
	for name, prop := range s.Properties {
		if prop == nil {
			continue
		}
		fields = append(fields, respField{
			GoName:   paramNameToGoName(name),
			APIName:  name,
			GoType:   goTypeForSchema(prop),
			Comment:  oneLine(prop.Description),
			Required: requiredSet[name],
		})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].APIName < fields[j].APIName })
	return fields
}

// goTypeForSchema resolves a schema node to a Go type name.
//
// Refs to objs_* become the corresponding Go type (e.g. "*Message").
// Refs to defs_* become the underlying primitive Go type.
// oneOf/anyOf/unmodelled shapes degrade gracefully to json.RawMessage.
func goTypeForSchema(s *schema) string {
	if s == nil {
		return "json.RawMessage"
	}
	if s.Ref != "" {
		return resolveRefToGoType(s.Ref)
	}
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		return "json.RawMessage"
	}
	switch s.typeStr() {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if inner := s.itemsSchema(); inner != nil {
			return "[]" + goTypeForSchema(inner)
		}
		// Tuple form — collapse to []any.
		return "[]any"
	case "object":
		// Inline object — emit as map[string]any rather than synthesising
		// an anonymous struct. Anonymous structs are uglier in callers' code.
		return "map[string]any"
	case "":
		// Slack omits "type" for nullable-scalar fields, signalling
		// nullability via items=[{$ref: defs_X}, {type:"null"}]. Peel
		// that back to the underlying primitive type rather than emit a
		// raw json.RawMessage that callers have to manually unmarshal.
		if tuple := s.itemsTuple(); len(tuple) > 0 {
			if hit := firstNonNull(tuple); hit != nil {
				return goTypeForSchema(hit)
			}
		}
		// Truly untyped — fall back to raw JSON.
		return "json.RawMessage"
	default:
		return "json.RawMessage"
	}
}

// resolveRefToGoType converts a $ref like "#/definitions/objs_message" to a
// Go type expression. Pointer for object types (so nil-checks work cleanly
// for optional response fields), primitive for `defs_*` aliases.
func resolveRefToGoType(ref string) string {
	const prefix = "#/definitions/"
	if !strings.HasPrefix(ref, prefix) {
		return "json.RawMessage"
	}
	name := ref[len(prefix):]

	// Primitive defs alias: emit underlying Go type.
	if strings.HasPrefix(name, "defs_") {
		if def, ok := globalSpec.Definitions[name]; ok {
			t := def.typeStr()
			if t != "object" {
				switch t {
				case "string":
					return "string"
				case "integer":
					return "int"
				case "number":
					return "float64"
				case "boolean":
					return "bool"
				default:
					return "json.RawMessage"
				}
			}
		}
		return "json.RawMessage"
	}

	// objs_* / blocks → Go struct pointer, but only if the target was actually
	// emitted (has type=object + properties). Loose / empty / array-typed
	// definitions degrade to json.RawMessage so generated code stays valid.
	if def, ok := globalSpec.Definitions[name]; ok {
		if def.typeStr() == "array" {
			// Heterogeneous block lists, etc.
			return "[]json.RawMessage"
		}
		if def.typeStr() != "object" || len(def.Properties) == 0 {
			return "json.RawMessage"
		}
	}
	return "*" + defNameToGoName(name)
}

// ─── method extraction ─────────────────────────────────────────────────────

func extractMethods(s *spec) []method {
	methods := make([]method, 0, len(s.Paths))
	for path, verbs := range s.Paths {
		apiName := strings.TrimPrefix(path, "/")

		var verb string
		var op operation
		if v, ok := verbs["post"]; ok {
			verb = "POST"
			op = v
		} else if v, ok := verbs["get"]; ok {
			verb = "GET"
			op = v
		} else {
			continue
		}

		m := method{
			APIName:  apiName,
			GoName:   apiNameToGoName(apiName),
			HTTPVerb: verb,
			Summary:  strings.TrimSpace(op.Summary),
			Scopes:   extractScopes(op.Security),
		}
		m.Params, m.HasNonTokenParams = extractParams(op.Parameters)

		// Walk response.200 schema for typed response fields.
		if r, ok := op.Responses["200"]; ok && r.Schema != nil {
			m.RespFields = schemaFields(r.Schema)
		}

		methods = append(methods, m)
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].APIName < methods[j].APIName })
	return methods
}

func extractScopes(security []map[string][]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range security {
		for _, scopes := range s {
			for _, sc := range scopes {
				if seen[sc] {
					continue
				}
				seen[sc] = true
				out = append(out, sc)
			}
		}
	}
	sort.Strings(out)
	return out
}

func extractParams(params []param) ([]paramInfo, bool) {
	var infos []paramInfo
	hasNonToken := false
	for _, p := range params {
		if p.Name == "token" {
			continue
		}
		if p.In == "body" {
			continue
		}
		hasNonToken = true
		infos = append(infos, paramInfo{
			GoName:    paramNameToGoName(p.Name),
			APIName:   p.Name,
			GoType:    paramGoType(p),
			Required:  p.Required,
			Comment:   oneLine(p.Description),
			URLEncode: paramURLTag(p),
		})
	}
	return infos, hasNonToken
}

func paramURLTag(p param) string {
	tag := p.Name
	if !p.Required {
		tag += ",omitempty"
	}
	return fmt.Sprintf(`url:%q`, tag)
}

// timestampParams are mis-typed in Slack's OpenAPI as `number` but are
// always sent/received as strings (e.g. "1706000000.123456"). Encoding
// them as float64 loses sub-second precision Slack relies on for message
// identity, so override them to string here.
var timestampParams = map[string]bool{
	"ts": true, "thread_ts": true, "latest": true, "oldest": true,
	"timestamp": true, "update_ts": true, "message_ts": true,
}

func paramGoType(p param) string {
	if timestampParams[p.Name] {
		return "string"
	}
	switch p.Type {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if p.Items != nil {
			switch p.Items.Type {
			case "string":
				return "[]string"
			case "integer":
				return "[]int"
			}
		}
		return "[]string"
	default:
		return "string"
	}
}

// ─── name munging ──────────────────────────────────────────────────────────

func apiNameToGoName(apiName string) string {
	parts := strings.Split(apiName, ".")
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(camelInitUpper(p))
	}
	return b.String()
}

// defNameToGoName converts an OpenAPI definition name to its Go type name.
//
//	objs_message       → Message
//	objs_user_profile  → UserProfile
//	objs_bot_profile   → BotProfile
//	blocks             → Blocks
func defNameToGoName(s string) string {
	s = strings.TrimPrefix(s, "objs_")
	s = strings.TrimPrefix(s, "defs_")
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(camelInitUpper(p))
	}
	out := b.String()
	if out == "" {
		return "Field"
	}
	return out
}

func paramNameToGoName(p string) string {
	parts := strings.FieldsFunc(p, func(r rune) bool { return r == '_' || r == '.' || r == '-' })
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(camelInitUpper(part))
	}
	out := b.String()
	if out == "" {
		return "Field"
	}
	// Note: no keyword-collision suffix needed. After camelInitUpper the
	// name is upper-camelcase; Go keywords are all lowercase, so a
	// generated field name can never collide. Past versions appended `_`
	// for builtin-typename overlaps (Type, Range, Map, etc.) — those are
	// legal as field names and don't need escaping.
	return out
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

var commonAcronym = map[string]bool{
	"ID": true, "URL": true, "API": true, "TS": true, "UID": true,
	"DM": true, "IP": true, "JSON": true, "CSV": true, "MPIM": true,
	"TZ": true, "MS": true, "OK": true, "OAUTH": true,
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 140 {
		s = s[:140] + "..."
	}
	return s
}

// ─── writers ───────────────────────────────────────────────────────────────

const headerComment = `// Code generated by cmd/gen-api from Slack's Web API OpenAPI 2.0 spec.
// DO NOT EDIT — regenerate with ` + "`make api-gen`" + `.
`

var tmplFuncs = template.FuncMap{
	"join": strings.Join,
}

var typesTmpl = template.Must(template.New("types").Funcs(tmplFuncs).Parse(headerComment + `
package slackapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Ensure all imports are used regardless of which methods are emitted.
var (
	_ = json.RawMessage{}
	_ = url.Values{}
	_ = strconv.Itoa
	_ = strings.Builder{}
	_ = fmt.Sprintf
)

// ─── Shared object types (from spec's definitions/*) ──────────────────────
{{range .Defs}}
{{if .Title}}// {{.GoName}} — {{.Title}}.
{{end -}}
type {{.GoName}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} ` + "`json:\"{{.APIName}},omitempty\"`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}
{{end}}

// ─── Per-method params + responses ────────────────────────────────────────
{{range .Methods}}
{{if .HasNonTokenParams}}
// {{.GoName}}Params holds the parameters for the {{.APIName}} method.
//
// {{.Summary}}
type {{.GoName}}Params struct {
{{- range .Params}}
	{{.GoName}} {{.GoType}} ` + "`{{.URLEncode}}`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}

// values renders the params as form-encoded url.Values, suitable for POST
// to https://slack.com/api/{{.APIName}}.
func (p *{{.GoName}}Params) values() url.Values {
	v := url.Values{}
	if p == nil { return v }
{{- range .Params}}
{{- if eq .GoType "string"}}
	{{if not .Required}}if p.{{.GoName}} != "" { {{end}}v.Set({{printf "%q" .APIName}}, p.{{.GoName}}){{if not .Required}} }{{end}}
{{- else if eq .GoType "int"}}
	{{if not .Required}}if p.{{.GoName}} != 0 { {{end}}v.Set({{printf "%q" .APIName}}, strconv.Itoa(p.{{.GoName}})){{if not .Required}} }{{end}}
{{- else if eq .GoType "float64"}}
	{{if not .Required}}if p.{{.GoName}} != 0 { {{end}}v.Set({{printf "%q" .APIName}}, strconv.FormatFloat(p.{{.GoName}}, 'g', -1, 64)){{if not .Required}} }{{end}}
{{- else if eq .GoType "bool"}}
	{{if not .Required}}if p.{{.GoName}} { {{end}}v.Set({{printf "%q" .APIName}}, strconv.FormatBool(p.{{.GoName}})){{if not .Required}} }{{end}}
{{- else if eq .GoType "[]string"}}
	if len(p.{{.GoName}}) > 0 { v.Set({{printf "%q" .APIName}}, strings.Join(p.{{.GoName}}, ",")) }
{{- else if eq .GoType "[]int"}}
	if len(p.{{.GoName}}) > 0 {
		parts := make([]string, len(p.{{.GoName}}))
		for i, n := range p.{{.GoName}} { parts[i] = strconv.Itoa(n) }
		v.Set({{printf "%q" .APIName}}, strings.Join(parts, ","))
	}
{{- end}}
{{- end}}
	return v
}
{{end}}
// {{.GoName}}Response is the typed response envelope for {{.APIName}}.
//
// Top-level fields are walked from the spec's response.200 schema. The
// embedded BaseResponse carries ok/error; Raw retains the full body for
// callers needing fields the spec doesn't model.
type {{.GoName}}Response struct {
	BaseResponse
{{- range .RespFields}}
{{- if and (ne .APIName "ok") (ne .APIName "error") (ne .APIName "warning") (ne .APIName "response_metadata")}}
	{{.GoName}} {{.GoType}} ` + "`json:\"{{.APIName}},omitempty\"`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
{{- end}}
	Raw json.RawMessage ` + "`json:\"-\"`" + ` // full response body, populated by the operation wrapper
}

func (r *{{.GoName}}Response) envelope() *BaseResponse { return &r.BaseResponse }
{{end}}
`))

var operationsTmpl = template.Must(template.New("operations").Funcs(tmplFuncs).Parse(headerComment + `
package slackapi

import (
	"context"
	"encoding/json"
	"net/url"
)

// Ensure imports are used.
var (
	_ = url.Values{}
	_ = context.Background
	_ = json.RawMessage{}
)
{{range .}}
// {{.GoName}} calls Slack's {{.APIName}} method.
//
// {{.Summary}}
//
// Required scopes (any one combination): {{join .Scopes ", "}}
func {{.GoName}}(ctx context.Context, c *Client{{if .HasNonTokenParams}}, params *{{.GoName}}Params{{end}}) (*{{.GoName}}Response, error) {
	var resp {{.GoName}}Response
{{- if .HasNonTokenParams}}
	form := params.values()
{{- else}}
	form := url.Values{}
{{- end}}
	body, err := Do(ctx, c, "{{.APIName}}", form, &resp)
	resp.Raw = body
	if err != nil {
		return &resp, err
	}
	return &resp, nil
}
{{end}}
`))

var scopesTmpl = template.Must(template.New("scopes").Funcs(tmplFuncs).Parse(headerComment + `
package slackapi

// MethodScopes maps every Slack Web API method to the OAuth scopes Slack
// requires (any one of which is sufficient — Slack's spec lists them as a
// single union via security[0].slackAuth).
//
// Generated from api/specs/slack_web.json.
var MethodScopes = map[string][]string{
{{range .}}	"{{.APIName}}": { {{range $i, $s := .Scopes}}{{if $i}}, {{end}}{{printf "%q" $s}}{{end}} },
{{end}}}

// AllScopes returns every scope referenced by any method, deduplicated.
func AllScopes() []string {
	seen := map[string]bool{}
	out := make([]string, 0, 64)
	for _, scopes := range MethodScopes {
		for _, s := range scopes {
			if seen[s] { continue }
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ScopesForMethods returns the union of scopes for the given methods.
func ScopesForMethods(methods []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range methods {
		for _, s := range MethodScopes[m] {
			if seen[s] { continue }
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
`))

func writeTypes(path string, methods []method, defs []defType) error {
	return executeTemplate(path, typesTmpl, struct {
		Methods []method
		Defs    []defType
	}{Methods: methods, Defs: defs})
}

func writeOperations(path string, methods []method) error {
	return executeTemplate(path, operationsTmpl, methods)
}

func writeScopes(path string, methods []method) error {
	return executeTemplate(path, scopesTmpl, methods)
}

func executeTemplate(path string, t *template.Template, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}
