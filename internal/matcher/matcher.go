package matcher

import "strings"

// MatchResult holds the outcome of matching patterns against repo slugs.
type MatchResult struct {
	Matched   []string // deduplicated slugs that matched at least one pattern
	Unmatched []string // patterns that matched zero slugs
}

// Match checks each pattern against all slugs using case-insensitive substring matching.
// Space-separated terms within a pattern use AND logic (all must appear in slug).
func Match(slugs []string, patterns []string) MatchResult {
	seen := make(map[string]bool)
	var matched []string
	var unmatched []string

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		terms := strings.Fields(strings.ToLower(pattern))
		found := false

		for _, slug := range slugs {
			if matchTerms(strings.ToLower(slug), terms) {
				if !seen[slug] {
					seen[slug] = true
					matched = append(matched, slug)
				}
				found = true
			}
		}

		if !found {
			unmatched = append(unmatched, pattern)
		}
	}

	return MatchResult{Matched: matched, Unmatched: unmatched}
}

// matchTerms returns true if all terms are substrings of slug.
func matchTerms(slug string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(slug, t) {
			return false
		}
	}
	return true
}
