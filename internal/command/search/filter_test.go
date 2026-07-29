package search

import (
	"reflect"
	"testing"
)

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]interface{}
	}{
		{
			name:  "no filters yields nil so the key is omitted entirely",
			input: nil,
			want:  nil,
		},
		{
			name:  "exact match",
			input: []string{"issn=1664-302X"},
			want:  map[string]interface{}{"issn": "1664-302X"},
		},
		{
			name:  "comma separates into any-of",
			input: []string{"issn=1664-302X,1932-6203"},
			want: map[string]interface{}{
				"issn": []interface{}{"1664-302X", "1932-6203"},
			},
		},
		{
			name:  "exclusion uses the API's own not form",
			input: []string{"issn!=1932-6203"},
			want: map[string]interface{}{
				"issn": map[string]interface{}{"not": "1932-6203"},
			},
		},
		{
			name:  "exclusion of several",
			input: []string{"publisher!=Elsevier,Springer"},
			want: map[string]interface{}{
				"publisher": map[string]interface{}{
					"not": []interface{}{"Elsevier", "Springer"},
				},
			},
		},
		{
			name:  "metric threshold emits a number, not a string",
			input: []string{"journal_metric.2yr_mean_citedness>=5"},
			want: map[string]interface{}{
				"journal_metric.2yr_mean_citedness": map[string]interface{}{"gte": 5.0},
			},
		},
		{
			name:  "strict less-than",
			input: []string{"journal_metric.2yr_mean_citedness<2.5"},
			want: map[string]interface{}{
				"journal_metric.2yr_mean_citedness": map[string]interface{}{"lt": 2.5},
			},
		},
		{
			name:  "date range stays a string for the server to detect",
			input: []string{"publication_date>=2020-01-01"},
			want: map[string]interface{}{
				"publication_date": map[string]interface{}{"gte": "2020-01-01"},
			},
		},
		{
			name:  "several filters are ANDed as separate keys",
			input: []string{"issn=1664-302X", "publisher=SAGE Publications"},
			want: map[string]interface{}{
				"issn":      "1664-302X",
				"publisher": "SAGE Publications",
			},
		},
		{
			name:  "doi keeps its case for the server to normalize",
			input: []string{"doi=10.1345/APH.1G425"},
			want:  map[string]interface{}{"doi": "10.1345/APH.1G425"},
		},
		{
			name:  "value containing = survives (DOIs and queries can hold one)",
			input: []string{"doi=10.1/a=b"},
			want:  map[string]interface{}{"doi": "10.1/a=b"},
		},
		{
			name:  "trailing comma does not produce an empty term",
			input: []string{"issn=1664-302X,"},
			want:  map[string]interface{}{"issn": "1664-302X"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilters(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFilters(%q)\n got: %#v\nwant: %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFiltersErrors(t *testing.T) {
	for _, input := range [][]string{
		{"nooperator"},
		{"=novalue"},
		{"nokey="},
		{"issn=a", "issn=b"}, // ambiguous: would silently drop one
	} {
		t.Run(input[0], func(t *testing.T) {
			if _, err := ParseFilters(input); err == nil {
				t.Errorf("ParseFilters(%q) = nil error, want error", input)
			}
		})
	}
}

func TestParseFilterJSON(t *testing.T) {
	got, err := ParseFilterJSON(`{"or":[{"field":"issn","eq":"1664-302X"}]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["or"]; !ok {
		t.Errorf("expected an 'or' combinator, got %#v", got)
	}
}

func TestParseFilterJSONEmptyIsNil(t *testing.T) {
	got, err := ParseFilterJSON("   ")
	if err != nil || got != nil {
		t.Errorf("ParseFilterJSON(blank) = (%#v, %v), want (nil, nil)", got, err)
	}
}

func TestParseFilterJSONRejectsGarbage(t *testing.T) {
	if _, err := ParseFilterJSON("not json"); err == nil {
		t.Error("expected an error for non-JSON input")
	}
}
