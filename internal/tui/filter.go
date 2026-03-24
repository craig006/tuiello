package tui

import (
	"strings"

	"github.com/craig006/tuillo/internal/trello"
)

// Filter holds the parsed filter state.
type Filter struct {
	Text    string
	Members []string
	Labels  []string
}

// IsEmpty returns true if no filters are active.
func (f Filter) IsEmpty() bool {
	return f.Text == "" && len(f.Members) == 0 && len(f.Labels) == 0
}

// ParseFilter parses a search string into structured filter components.
// Recognized tokens: member:<value>, label:<value>. Quoted values supported.
// Everything else becomes the text search.
func ParseFilter(input string, currentUser string) Filter {
	var f Filter
	var textParts []string

	tokens := tokenize(input)
	for _, tok := range tokens {
		lower := strings.ToLower(tok)
		if strings.HasPrefix(lower, "member:") {
			val := tok[len("member:"):]
			val = strings.Trim(val, `"`)
			if val == "@me" && currentUser != "" {
				val = currentUser
			}
			if val != "" {
				f.Members = append(f.Members, val)
			}
		} else if strings.HasPrefix(lower, "label:") {
			val := tok[len("label:"):]
			val = strings.Trim(val, `"`)
			if val != "" {
				f.Labels = append(f.Labels, val)
			}
		} else {
			textParts = append(textParts, tok)
		}
	}

	f.Text = strings.TrimSpace(strings.Join(textParts, " "))
	return f
}

// tokenize splits input into tokens, respecting quoted values after member:/label: prefixes.
func tokenize(input string) []string {
	var tokens []string
	i := 0
	runes := []rune(input)
	for i < len(runes) {
		// Skip whitespace
		if runes[i] == ' ' {
			i++
			continue
		}
		start := i
		// Check for member: or label: prefix with quoted value
		rest := string(runes[i:])
		lowerRest := strings.ToLower(rest)
		if strings.HasPrefix(lowerRest, "member:\"") || strings.HasPrefix(lowerRest, "label:\"") {
			colonIdx := strings.Index(rest, ":")
			i += colonIdx + 1 // past the colon
			if i < len(runes) && runes[i] == '"' {
				i++ // past opening quote
				end := i
				for end < len(runes) && runes[end] != '"' {
					end++
				}
				prefix := string(runes[start : start+colonIdx+1])
				val := string(runes[i:end])
				tokens = append(tokens, prefix+`"`+val+`"`)
				if end < len(runes) {
					end++ // past closing quote
				}
				i = end
				continue
			}
		}
		// Regular token (until next space)
		for i < len(runes) && runes[i] != ' ' {
			i++
		}
		tokens = append(tokens, string(runes[start:i]))
	}
	return tokens
}

// MatchesCard returns true if the card passes all active filters.
func (f Filter) MatchesCard(card trello.Card) bool {
	if f.IsEmpty() {
		return true
	}

	// Text: case-insensitive substring match on card name
	if f.Text != "" {
		if !strings.Contains(strings.ToLower(card.Name), strings.ToLower(f.Text)) {
			return false
		}
	}

	// Members: card must have at least one matching member (OR)
	if len(f.Members) > 0 {
		found := false
		for _, fm := range f.Members {
			for _, cm := range card.Members {
				if strings.EqualFold(cm.Username, fm) || strings.EqualFold(cm.FullName, fm) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Labels: card must have at least one matching label (OR)
	if len(f.Labels) > 0 {
		found := false
		for _, fl := range f.Labels {
			for _, cl := range card.Labels {
				name := cl.Name
				if name == "" {
					name = cl.Color
				}
				if strings.EqualFold(name, fl) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// BuildFilterText reconstructs the search bar text from a Filter.
func BuildFilterText(f Filter) string {
	var parts []string
	if f.Text != "" {
		parts = append(parts, f.Text)
	}
	for _, m := range f.Members {
		if strings.Contains(m, " ") {
			parts = append(parts, `member:"`+m+`"`)
		} else {
			parts = append(parts, "member:"+m)
		}
	}
	for _, l := range f.Labels {
		if strings.Contains(l, " ") {
			parts = append(parts, `label:"`+l+`"`)
		} else {
			parts = append(parts, "label:"+l)
		}
	}
	return strings.Join(parts, " ")
}
