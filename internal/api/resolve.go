package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
)

// Resolver maps human-friendly channel names (#general) and user names (@alice)
// to their Slack IDs. Results are cached for the lifetime of the process.
type Resolver struct {
	client      *slackapi.Client
	mu          sync.Mutex
	channels    map[string]string // name → ID
	users       map[string]string // name → ID
	loaded      bool
	usersLoaded bool
}

// NewResolver creates a new Resolver backed by the given Slack client.
func NewResolver(client *slackapi.Client) *Resolver {
	return &Resolver{
		client:   client,
		channels: make(map[string]string),
		users:    make(map[string]string),
	}
}

// ResolveChannel takes a channel name (with or without #) or a channel ID
// and returns the channel ID.
func (r *Resolver) ResolveChannel(nameOrID string) (string, error) {
	// Strip leading # if present
	nameOrID = strings.TrimPrefix(nameOrID, "#")

	// If it looks like a channel ID (starts with C, G, or D), return as-is
	if isChannelID(nameOrID) {
		return nameOrID, nil
	}

	r.mu.Lock()
	if id, ok := r.channels[nameOrID]; ok {
		r.mu.Unlock()
		return id, nil
	}
	r.mu.Unlock()

	// Load channels if not yet cached
	if err := r.loadChannels(); err != nil {
		return "", fmt.Errorf("failed to resolve channel %q: %w", nameOrID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.channels[nameOrID]; ok {
		return id, nil
	}

	// No exact match — try fuzzy. `@michell` → `@michelle`, `#stand` → `#stand-up`.
	candidates := mapKeys(r.channels)
	if matched, tier, ok := fuzzyMatch(nameOrID, candidates); ok && tier != matchExact {
		// Only accept fuzzy hits with a confidence level above raw fuzzy —
		// substring/contains is fine, RankMatch alone is too risky for
		// channel writes (you don't want to post to the wrong channel).
		if tier == matchContains {
			return r.channels[matched], nil
		}
	}

	suggestions := suggestSimilar(nameOrID, candidates, 3)
	if len(suggestions) > 0 {
		return "", fmt.Errorf("channel %q not found. Did you mean: %s?", nameOrID, strings.Join(suggestions, ", "))
	}
	return "", fmt.Errorf("channel %q not found", nameOrID)
}

// ResolveUser takes a username (with or without @) or a user ID and returns the user ID.
func (r *Resolver) ResolveUser(nameOrID string) (string, error) {
	// Strip leading @ if present
	nameOrID = strings.TrimPrefix(nameOrID, "@")

	// If it looks like a user ID (starts with U or W), return as-is
	if isUserID(nameOrID) {
		return nameOrID, nil
	}

	r.mu.Lock()
	if id, ok := r.users[nameOrID]; ok {
		r.mu.Unlock()
		return id, nil
	}
	r.mu.Unlock()

	// Load users if not yet cached
	if err := r.loadUsers(); err != nil {
		return "", fmt.Errorf("failed to resolve user %q: %w", nameOrID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.users[nameOrID]; ok {
		return id, nil
	}

	// No exact match — accept fuzzy contains (`@michell` → `@michelle`).
	// Don't accept raw fuzzy here: typing the wrong handle and DM-ing the
	// wrong person is worse than rejecting the input.
	candidates := mapKeys(r.users)
	if matched, tier, ok := fuzzyMatch(nameOrID, candidates); ok && tier == matchContains {
		return r.users[matched], nil
	}

	suggestions := suggestSimilar(nameOrID, candidates, 3)
	if len(suggestions) > 0 {
		return "", fmt.Errorf("user %q not found. Did you mean: %s?", nameOrID, strings.Join(suggestions, ", "))
	}
	return "", fmt.Errorf("user %q not found", nameOrID)
}

// mapKeys returns a deduplicated list of map keys.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (r *Resolver) loadChannels() error {
	r.mu.Lock()
	if r.loaded {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	params := &slackapi.ConversationsListParams{
		Types:           "public_channel,private_channel,im,mpim",
		Limit:           1000,
		ExcludeArchived: true,
	}

	ctx := context.Background()
	for {
		resp, err := slackapi.ConversationsList(ctx, r.client, params)
		if err != nil {
			return err
		}
		r.mu.Lock()
		for _, ch := range resp.Channels {
			if ch == nil {
				continue
			}
			r.channels[ch.Name] = ch.ID
		}
		r.mu.Unlock()
		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		params.Cursor = resp.ResponseMetadata.NextCursor
	}

	r.mu.Lock()
	r.loaded = true
	r.mu.Unlock()

	return nil
}

// ResolveDM takes a username, @username, or user ID and returns a DM channel ID.
// It resolves the user and opens (or retrieves) a DM conversation via conversations.open.
func (r *Resolver) ResolveDM(nameOrID string) (string, error) {
	userID, err := r.ResolveUser(nameOrID)
	if err != nil {
		return "", err
	}

	resp, err := slackapi.ConversationsOpen(context.Background(), r.client, &slackapi.ConversationsOpenParams{
		Users: userID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open DM with %q: %w", nameOrID, err)
	}
	if resp.Channel == nil || resp.Channel.ID == "" {
		return "", fmt.Errorf("open DM with %q: response had no channel ID", nameOrID)
	}
	return resp.Channel.ID, nil
}

// LooksLikeUser returns true if the target looks like a user reference
// (starts with @, starts with U/W, or is not #-prefixed and not a channel ID).
func LooksLikeUser(target string) bool {
	if strings.HasPrefix(target, "@") {
		return true
	}
	if isUserID(target) {
		return true
	}
	if strings.HasPrefix(target, "#") || isChannelID(target) {
		return false
	}
	// Bare name — assume user if it doesn't look like a channel ID
	return true
}

// ResolveTarget resolves a bare name to a channel ID, trying the most likely
// path first based on the name's syntax, then falling back to the other.
//
// Explicit prefixes are unambiguous:
//
//	#general → channel only (no fallback)
//	@alice   → DM only     (no fallback)
//
// Bare names (e.g. "stand-up", "herman") are ambiguous — the name could be
// either a channel or a user. We try the heuristic-preferred path first
// (user for bare names) and fall back to the other on failure.
//
// Returns the resolved channel ID and whether the target resolved as a DM.
func (r *Resolver) ResolveTarget(target string) (channelID string, isDM bool, err error) {
	// Explicit #channel — no ambiguity
	if strings.HasPrefix(target, "#") {
		id, err := r.ResolveChannel(target)
		return id, false, err
	}

	// Explicit @user or user ID — no ambiguity
	if strings.HasPrefix(target, "@") || isUserID(target) {
		id, err := r.ResolveDM(target)
		return id, true, err
	}

	// Explicit channel ID — no ambiguity
	if isChannelID(target) {
		return target, false, nil
	}

	// Bare name — ambiguous. Try user (DM) first, fall back to channel.
	id, dmErr := r.ResolveDM(target)
	if dmErr == nil {
		return id, true, nil
	}

	id, chErr := r.ResolveChannel(target)
	if chErr == nil {
		return id, false, nil
	}

	// Both failed — return a combined error
	return "", false, fmt.Errorf("could not resolve %q as a user or channel", target)
}

func (r *Resolver) loadUsers() error {
	r.mu.Lock()
	if r.usersLoaded {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	// Page through users.list. Slack returns up to 1000 per page; we
	// keep going until next_cursor empties.
	type userRecord struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Deleted bool   `json:"deleted"`
		Profile struct {
			DisplayName string `json:"display_name"`
		} `json:"profile"`
	}
	var allUsers []userRecord
	cursor := ""
	for {
		resp, err := slackapi.UsersList(context.Background(), r.client, &slackapi.UsersListParams{
			Limit:  1000,
			Cursor: cursor,
		})
		if err != nil {
			return err
		}
		var page struct {
			Members []userRecord `json:"members"`
		}
		if err := json.Unmarshal(resp.Raw, &page); err != nil {
			return err
		}
		allUsers = append(allUsers, page.Members...)
		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Track partial (first-name) candidates for second pass.
	// Key: lowercased first name, Value: set of user IDs that share it.
	type candidate struct {
		ids map[string]bool
	}
	partials := make(map[string]*candidate)

	for _, u := range allUsers {
		if u.Deleted {
			continue
		}
		r.users[u.Name] = u.ID
		lower := strings.ToLower(u.Name)
		if lower != u.Name {
			r.users[lower] = u.ID
		}
		if u.Profile.DisplayName != "" {
			r.users[u.Profile.DisplayName] = u.ID
			lowerDisplay := strings.ToLower(u.Profile.DisplayName)
			if lowerDisplay != u.Profile.DisplayName {
				r.users[lowerDisplay] = u.ID
			}
		}

		// Collect partial name from multi-word display names (e.g. "herman gorbatovskii" → "herman")
		if u.Profile.DisplayName != "" {
			if parts := strings.Fields(u.Profile.DisplayName); len(parts) > 1 {
				first := strings.ToLower(parts[0])
				if partials[first] == nil {
					partials[first] = &candidate{ids: make(map[string]bool)}
				}
				partials[first].ids[u.ID] = true
			}
		}
		// Collect partial name from dotted usernames (e.g. "herman.gorbatovskii" → "herman")
		if dotIdx := strings.IndexByte(lower, '.'); dotIdx > 0 {
			first := lower[:dotIdx]
			if partials[first] == nil {
				partials[first] = &candidate{ids: make(map[string]bool)}
			}
			partials[first].ids[u.ID] = true
		}
	}

	// Add unambiguous partial names (only when a single user owns the first name
	// and no exact-match key already exists).
	for name, cand := range partials {
		if len(cand.ids) != 1 {
			continue // ambiguous — multiple users share this first name
		}
		if _, exists := r.users[name]; exists {
			continue // an exact username/display-name already claims this key
		}
		for id := range cand.ids {
			r.users[name] = id
		}
	}

	r.usersLoaded = true
	return nil
}

func isChannelID(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[0] == 'C' || s[0] == 'G' || s[0] == 'D'
}

func isUserID(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[0] == 'U' || s[0] == 'W'
}

// isWordChar returns true if the byte is a letter, digit, underscore, hyphen, or period.
// These characters can appear in Slack usernames and display names.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.'
}

// ResolveMentions scans text for @name patterns and replaces resolved names
// with Slack's <@USERID> format. Unresolved names are left as-is.
// Returns the processed text, a list of resolved display names, and any API error.
func (r *Resolver) ResolveMentions(text string) (string, []string, error) {
	// Ensure users are loaded before scanning
	if err := r.loadUsers(); err != nil {
		return text, nil, err
	}

	// Build a sorted list of known names (longest first for greedy matching)
	r.mu.Lock()
	type sortedUser struct {
		lower string
		id    string
	}
	var sorted []sortedUser
	seen := make(map[string]bool) // dedupe by ID+lower to avoid double entries
	for name, id := range r.users {
		lower := strings.ToLower(name)
		key := id + "|" + lower
		if seen[key] {
			continue
		}
		seen[key] = true
		sorted = append(sorted, sortedUser{lower: lower, id: id})
	}
	r.mu.Unlock()

	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].lower) > len(sorted[j].lower)
	})

	// Find all @mention positions
	type mentionPos struct {
		start int
		end   int
		name  string
		id    string
	}

	textLower := strings.ToLower(text)
	var mentions []mentionPos

	for i := 0; i < len(text); i++ {
		if text[i] != '@' || i+1 >= len(text) {
			continue
		}
		// Skip if already inside a Slack mention like <@U...>
		if i > 0 && text[i-1] == '<' {
			continue
		}
		afterAt := textLower[i+1:]
		for _, su := range sorted {
			if !strings.HasPrefix(afterAt, su.lower) {
				continue
			}
			endPos := i + 1 + len(su.lower)
			if endPos < len(text) && isWordChar(text[endPos]) {
				continue
			}
			mentions = append(mentions, mentionPos{
				start: i,
				end:   endPos,
				name:  text[i+1 : endPos], // preserve original casing for display
				id:    su.id,
			})
			i = endPos - 1
			break
		}
	}

	if len(mentions) == 0 {
		return text, nil, nil
	}

	// Rebuild the text with Slack mention format
	var b strings.Builder
	var resolved []string
	pos := 0

	for _, m := range mentions {
		b.WriteString(text[pos:m.start])
		b.WriteString("<@")
		b.WriteString(m.id)
		b.WriteString(">")
		resolved = append(resolved, m.name)
		pos = m.end
	}
	b.WriteString(text[pos:])

	return b.String(), resolved, nil
}

// FindSimilarUsers returns usernames that are similar to the given name,
// using prefix matching and simple edit-distance heuristics.
// Useful for suggesting corrections when a mention doesn't resolve.
func (r *Resolver) FindSimilarUsers(name string) []string {
	_ = r.loadUsers() // best-effort load

	nameLower := strings.ToLower(name)

	r.mu.Lock()
	defer r.mu.Unlock()

	seen := make(map[string]bool)
	var matches []string

	for uname := range r.users {
		lower := strings.ToLower(uname)
		if seen[lower] {
			continue
		}

		// Prefix match
		if strings.HasPrefix(lower, nameLower) || strings.HasPrefix(nameLower, lower) {
			seen[lower] = true
			matches = append(matches, uname)
			continue
		}

		// Simple edit distance: if names differ by at most 2 characters and
		// are similar length, consider it a match.
		if abs(len(lower)-len(nameLower)) <= 2 && levenshtein(lower, nameLower) <= 2 {
			seen[lower] = true
			matches = append(matches, uname)
		}
	}

	sort.Strings(matches)
	if len(matches) > 5 {
		matches = matches[:5]
	}
	return matches
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = del
			if ins < curr[j] {
				curr[j] = ins
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
