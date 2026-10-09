package postgres

import (
	"reflect"
	"strings"
	"testing"
)

func TestExpandSynonyms(t *testing.T) {
	synonyms := map[string][]string{
		"sofa": {"couch", " settee ", ""},
		"Mug":  {"cup"},
	}
	cases := []struct {
		keyword string
		want    []string
	}{
		{"sofa", []string{"couch", "settee"}},
		{"SOFA", []string{"couch", "settee"}},
		{"  sofa  ", []string{"couch", "settee"}},
		{"mug", []string{"cup"}}, // keys are matched case-insensitively
		{"chair", nil},
		{"", nil},
	}
	for _, c := range cases {
		if got := expandSynonyms(c.keyword, synonyms); !reflect.DeepEqual(got, c.want) {
			t.Errorf("expandSynonyms(%q) = %v, want %v", c.keyword, got, c.want)
		}
	}
	if got := expandSynonyms("sofa", nil); got != nil {
		t.Errorf("no synonyms configured should return nil, got %v", got)
	}
}

func TestSearchClauseCoversStemmingSubstringAndSynonyms(t *testing.T) {
	where, args := searchClause("lamps", []string{"light"})

	// One stemmed match per term, plus the substring fallbacks.
	if n := strings.Count(where, "plainto_tsquery('english', ?)"); n != 2 {
		t.Errorf("expected a stemmed match for the keyword and each synonym, got %d: %s", n, where)
	}
	for _, fragment := range []string{"title ILIKE ?", "description ILIKE ?", "names::text ILIKE ?"} {
		if !strings.Contains(where, fragment) {
			t.Errorf("missing %s in %s", fragment, where)
		}
	}
	if strings.Contains(where, "%") && strings.Contains(where, "title % ?") {
		t.Error("whole-string similarity should not be part of the exact search")
	}

	// Keyword first, then the synonym, then the three LIKE patterns.
	want := []any{"lamps", "light", "%lamps%", "%lamps%", "%lamps%"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestFuzzyClauseUsesWordSimilarity(t *testing.T) {
	where, args := fuzzyClause("lampp")
	// The keyword must come first: word_similarity is directional and only
	// scores the coverage of its first argument.
	if !strings.Contains(where, "word_similarity(?::text, title::text)") ||
		!strings.Contains(where, "word_similarity(?::text, description::text)") {
		t.Errorf("fuzzy match should compare words, not whole strings: %s", where)
	}
	want := []any{"lampp", fuzzySimilarity, "lampp", fuzzySimilarity, "%lampp%"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
	// A threshold that is too strict would never match a single-letter typo.
	if fuzzySimilarity <= 0 || fuzzySimilarity > 0.6 {
		t.Errorf("fuzzySimilarity = %v is outside the useful range", fuzzySimilarity)
	}
}
