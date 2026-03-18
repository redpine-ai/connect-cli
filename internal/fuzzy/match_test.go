package fuzzy

import "testing"

func TestClosestMatch(t *testing.T) {
	tests := []struct {
		input      string
		candidates []string
		want       string
		wantFound  bool
	}{
		{"alleas", []string{"allears", "analytics", "billing"}, "allears", true},
		{"qurey", []string{"query", "delete", "list"}, "query", true},
		{"xyz", []string{"abc", "def"}, "", false},
		{"exact", []string{"exact", "other"}, "exact", true},
		{"", []string{"a", "b"}, "", false},
		{"test", []string{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, found := ClosestMatch(tt.input, tt.candidates)
			if found != tt.wantFound {
				t.Errorf("found=%v, want %v", found, tt.wantFound)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClosestMatches(t *testing.T) {
	matches := ClosestMatches("qurey", []string{"query", "queue", "delete"}, 2)
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	if matches[0] != "query" {
		t.Errorf("best match should be 'query', got %q", matches[0])
	}
}
