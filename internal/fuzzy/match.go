package fuzzy

import (
	"sort"

	"github.com/agnivade/levenshtein"
)

const maxDistance = 3

func ClosestMatch(input string, candidates []string) (string, bool) {
	matches := ClosestMatches(input, candidates, 1)
	if len(matches) == 0 {
		return "", false
	}
	return matches[0], true
}

func ClosestMatches(input string, candidates []string, limit int) []string {
	if input == "" || len(candidates) == 0 {
		return nil
	}

	type scored struct {
		name string
		dist int
	}

	var results []scored
	for _, c := range candidates {
		if c == input {
			return []string{c}
		}
		d := levenshtein.ComputeDistance(input, c)
		if d < maxDistance {
			results = append(results, scored{c, d})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	out := make([]string, 0, limit)
	for i, r := range results {
		if i >= limit {
			break
		}
		out = append(out, r.name)
	}
	return out
}
