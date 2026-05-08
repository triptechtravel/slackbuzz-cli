// Spec-quirk patches applied at load time.
//
// Slack's published OpenAPI 2.0 spec has a handful of consistent issues
// that hurt generated-code quality. Rather than work around them at
// every callsite, we patch the spec data once after parsing and before
// extraction. New patches go here, narrow and documented.
//
// Patch principles:
//
//   - Conservative: only fix things we've actively hit. Don't speculate.
//   - Reversible: a comment on each patch explaining what changed and why,
//     so future maintainers can re-evaluate when Slack ships a corrected spec.
//   - Single source: don't fix the same quirk both here and in
//     `goTypeForSchema` — keep the generator code generic.
package main

import (
	"encoding/json"
	"strings"
)

// applyPatches rewrites known spec quirks in-place.
func applyPatches(s *spec) {
	injectSyntheticDefinitions(s)
	patchInlineMessageArrays(s)
	patchKnownInlineRefs(s)
	patchDefinitionFieldRefs(s)
}

// injectSyntheticDefinitions adds Go-friendly object definitions for shapes
// Slack uses repeatedly inline but never names. Each entry is a minimal
// schema that the generator's normal path then turns into a typed Go
// struct, available for patches to ref.
func injectSyntheticDefinitions(s *spec) {
	if s.Definitions == nil {
		s.Definitions = map[string]schema{}
	}

	// Topic / purpose objects — Slack inlines `{value, creator, last_set}`
	// in objs_channel.topic and .purpose. Naming the shape lets us emit a
	// real Go type so callers don't fall back to map[string]any indexing.
	if _, exists := s.Definitions["objs_topic_purpose"]; !exists {
		s.Definitions["objs_topic_purpose"] = schema{
			Type:  json.RawMessage(`"object"`),
			Title: "Topic or Purpose Object (synthesised)",
			Properties: map[string]*schema{
				"value":    {Type: json.RawMessage(`"string"`)},
				"creator":  {Type: json.RawMessage(`"string"`)},
				"last_set": {Type: json.RawMessage(`"integer"`)},
			},
		}
	}

	// User object — Slack's spec defines objs_user as empty (no type, no
	// properties), so the generator skips it and refs to it fall back to
	// json.RawMessage. Synthesise a minimal-but-useful User type from the
	// fields commands actually consume.
	if def, exists := s.Definitions["objs_user"]; !exists || def.typeStr() == "" || len(def.Properties) == 0 {
		s.Definitions["objs_user"] = schema{
			Type:  json.RawMessage(`"object"`),
			Title: "User Object (synthesised — Slack's spec leaves objs_user empty)",
			Properties: map[string]*schema{
				"id":        {Type: json.RawMessage(`"string"`)},
				"team_id":   {Type: json.RawMessage(`"string"`)},
				"name":      {Type: json.RawMessage(`"string"`)},
				"deleted":   {Type: json.RawMessage(`"boolean"`)},
				"real_name": {Type: json.RawMessage(`"string"`)},
				"tz":        {Type: json.RawMessage(`"string"`)},
				"tz_label":  {Type: json.RawMessage(`"string"`)},
				"tz_offset": {Type: json.RawMessage(`"integer"`)},
				"is_bot":    {Type: json.RawMessage(`"boolean"`)},
				"is_admin":  {Type: json.RawMessage(`"boolean"`)},
				"is_owner":  {Type: json.RawMessage(`"boolean"`)},
				"updated":   {Type: json.RawMessage(`"integer"`)},
				"profile":   {Ref: "#/definitions/objs_user_profile"},
			},
		}
	}
}

// patchDefinitionFieldRefs rewrites properties inside `definitions/*` to
// `$ref` a sibling definition where the spec inlined an object that
// matches an existing definition. The result is properly typed Go fields
// instead of `map[string]any`.
//
// Currently used to upgrade `objs_channel.topic` and `.purpose` from
// loose maps to `*TopicPurposeCreator` (which is what `defs_topic_purpose_creator`
// generates as).
func patchDefinitionFieldRefs(s *spec) {
	patches := map[string]map[string]string{
		// definition name → property → target ref
		"objs_channel": {
			"topic":   "#/definitions/objs_topic_purpose",
			"purpose": "#/definitions/objs_topic_purpose",
		},
	}
	for defName, props := range patches {
		def, ok := s.Definitions[defName]
		if !ok || def.Properties == nil {
			continue
		}
		for prop, ref := range props {
			p, ok := def.Properties[prop]
			if !ok || p == nil {
				continue
			}
			p.Ref = ref
			p.Type = nil
			p.Properties = nil
			p.Items = nil
		}
		s.Definitions[defName] = def
	}
}

// patchKnownInlineRefs surgically replaces known-bad inline schemas with
// $ref pointers to the right definition. Add entries here when you discover
// a method whose response field could be typed but the spec inlines it.
//
// Maintained as a small allow-list rather than auto-detection because the
// spec's nesting forms vary too much to recognise generically.
func patchKnownInlineRefs(s *spec) {
	// path → response status → property → ref
	patches := map[string]map[string]map[string]string{
		"/conversations.replies": {
			"200": {"messages": "#/definitions/objs_message"},
		},
		// objs_conversation is empty in Slack's spec; objs_channel has the
		// fields callers actually want. DMs/MPIMs use a subset of the same
		// fields (id/is_im/etc.) so callers degrade gracefully.
		"/conversations.list": {
			"200": {"channels": "#/definitions/objs_channel"},
		},
		"/conversations.info": {
			"200": {"channel": "#/definitions/objs_channel"},
		},
		"/conversations.open": {
			"200": {"channel": "#/definitions/objs_channel"},
		},
		// objs_user is empty in Slack's spec; injectSyntheticDefinitions
		// adds a properties-bearing version. These patches point the
		// users.* methods at it so callers get *User typed responses.
		"/users.list": {
			"200": {"members": "#/definitions/objs_user"},
		},
		"/users.info": {
			"200": {"user": "#/definitions/objs_user"},
		},
	}
	for path, byStatus := range patches {
		op, ok := s.Paths[path]
		if !ok {
			continue
		}
		for verb, body := range op {
			for status, propMap := range byStatus {
				resp, ok := body.Responses[status]
				if !ok || resp.Schema == nil {
					continue
				}
				for prop, ref := range propMap {
					p, ok := resp.Schema.Properties[prop]
					if !ok || p == nil {
						continue
					}
					// Two cases:
					//   - existing type is "array" → wrap items as []*T
					//   - anything else → make it a direct $ref to *T
					if p.typeStr() == "array" {
						p.Items = json.RawMessage(`{"$ref":"` + ref + `"}`)
					} else {
						p.Ref = ref
						p.Type = nil // clear so $ref takes effect
						p.Properties = nil
						p.Items = nil
					}
				}
				body.Responses[status] = resp
			}
			op[verb] = body
		}
		s.Paths[path] = op
	}
}

// patchInlineMessageArrays rewrites response schema properties named
// "messages" whose `items` is an inline object schema, replacing it with a
// `$ref` to objs_message. The spec uses inline objects in some methods
// (conversations.replies) but a $ref in others (conversations.history) for
// what is conceptually the same shape — the inline form generates as
// `[]map[string]any`, useless for callers wanting typed access.
//
// Limitation: this only patches at the response schema's top-level
// "messages" property. Deeper nestings aren't rewritten.
func patchInlineMessageArrays(s *spec) {
	const objsMessageRef = `{"$ref":"#/definitions/objs_message"}`

	for _, verbs := range s.Paths {
		for _, op := range verbs {
			for status, resp := range op.Responses {
				if resp.Schema == nil || resp.Schema.Properties == nil {
					continue
				}
				prop, ok := resp.Schema.Properties["messages"]
				if !ok || prop == nil {
					continue
				}
				if prop.typeStr() != "array" {
					continue
				}
				// Only rewrite when items is an inline object (no ref) or
				// a tuple of inline objects.
				if hasObjectItems(prop) {
					prop.Items = json.RawMessage(objsMessageRef)
					op.Responses[status] = resp
				}
			}
		}
	}
}

// hasObjectItems returns true when items is an inline object schema —
// either a single `{"type":"object",...}` or a tuple containing one.
func hasObjectItems(s *schema) bool {
	if len(s.Items) == 0 {
		return false
	}
	// Skip if already a ref.
	trimmed := skipWhitespace(s.Items)
	if strings.Contains(string(trimmed), `"$ref"`) && !strings.Contains(string(trimmed), `"$ref":""`) {
		return false
	}
	// Inline single object.
	var single schema
	if err := json.Unmarshal(s.Items, &single); err == nil {
		if single.typeStr() == "object" || (single.typeStr() == "" && len(single.Properties) > 0) {
			return true
		}
	}
	return false
}

func skipWhitespace(b []byte) []byte {
	for i, c := range b {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		}
		return b[i:]
	}
	return b
}
