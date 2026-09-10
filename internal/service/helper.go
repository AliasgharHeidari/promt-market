package service

import "github.com/lib/pq"

// ============================================================
//  HELPERS
// ============================================================

// pqStringArray converts a Go []string into the pq.StringArray type used
// by the domain model for Postgres text[] columns. Mirrors what the
// prompt service does when creating/updating prompts.
func pqStringArray(items []string) pq.StringArray {
	return pq.StringArray(items)
}

// truncate shortens text to maxLen characters, appending "..." when the
// input is longer than the limit. Used for SEO meta descriptions.
func truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}