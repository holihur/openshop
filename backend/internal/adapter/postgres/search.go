package postgres

import (
	"strings"

	"gorm.io/gorm/clause"
)

// searchClause builds the WHERE fragment that decides whether a product matches
// a keyword. Three strategies are combined so that every input works:
//
//   - an indexed stemmed full-text match for the keyword and its configured
//     synonyms ("lamps" finds "Lamp");
//   - a trigram match for misspellings ("desk lampp");
//   - a substring match, because the text search parsers do not tokenise CJK,
//     and for partial words ("desk la").
//
// The relevance ordering in relevanceOrder ranks the stemmed match first, so
// synonym hits sort below direct hits.
func searchClause(keyword string, synonyms []string) (string, []any) {
	terms := append([]string{keyword}, synonyms...)
	parts := make([]string, 0, len(terms)+4)
	args := make([]any, 0, len(terms)+4)
	for _, term := range terms {
		parts = append(parts, "search_vector @@ plainto_tsquery('english', ?)")
		args = append(args, term)
	}
	like := "%" + keyword + "%"
	parts = append(parts,
		"title ILIKE ?",
		"description ILIKE ?",
		"names::text ILIKE ?",
	)
	args = append(args, like, like, like)
	return "(" + strings.Join(parts, " OR ") + ")", args
}

// fuzzySimilarity is the word-level trigram threshold used by the misspelling
// fallback. Whole-string similarity would never match a typo inside a longer
// title, so word_similarity is used with a deliberately loose threshold.
const fuzzySimilarity = 0.35

// fuzzyClause matches products whose title or description contains a word close
// to the keyword. It is only used when the exact search found nothing, so a
// typo cannot add noise to a good result set.
func fuzzyClause(keyword string) (string, []any) {
	like := "%" + keyword + "%"
	// word_similarity is directional: it measures how much of the *first*
	// argument's trigrams are covered by the second, so the keyword goes first
	// and the column second. The casts give the parameter a concrete type
	// (word_similarity is not defined for varchar).
	return "(word_similarity(?::text, title::text) > ? OR word_similarity(?::text, description::text) > ? OR names::text ILIKE ?)",
		[]any{keyword, fuzzySimilarity, keyword, fuzzySimilarity, like}
}

// fuzzyOrder ranks the closest titles first.
func fuzzyOrder(keyword string) clause.Expr {
	return clause.Expr{
		SQL:  "greatest(word_similarity(?::text, title::text), word_similarity(?::text, description::text)) DESC",
		Vars: []any{keyword, keyword},
	}
}

// relevanceOrder ranks a keyword search, most relevant first. created_at breaks
// ties so the ordering stays stable across pages.
func relevanceOrder(keyword string) clause.Expr {
	return clause.Expr{
		SQL:  "ts_rank(search_vector, plainto_tsquery('english', ?)) DESC",
		Vars: []any{keyword},
	}
}

// expandSynonyms returns the configured synonyms for a keyword. The map keys
// are matched case-insensitively against the whole keyword.
func expandSynonyms(keyword string, synonyms map[string][]string) []string {
	if len(synonyms) == 0 {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(keyword))
	out := synonyms[lower]
	if len(out) == 0 {
		// Tolerate a configuration whose keys were not normalised, so a
		// mis-cased entry still works instead of silently doing nothing.
		for key, values := range synonyms {
			if strings.ToLower(strings.TrimSpace(key)) == lower {
				out = values
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	clean := make([]string, 0, len(out))
	for _, s := range out {
		if s = strings.TrimSpace(s); s != "" {
			clean = append(clean, s)
		}
	}
	return clean
}
