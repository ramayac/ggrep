package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestSearcher_ScanStream(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		regex        string
		contextLines int
		expected     []string
	}{
		{
			name:         "Simple match",
			input:        "foo\nbar\nbaz",
			regex:        "bar",
			contextLines: 1,
			expected:     []string{"    test.txt:2:bar"},
		},
		{
			name:         "Match with context",
			input:        "line1\nline2\nline3\nline4",
			regex:        "line2",
			contextLines: 2,
			expected: []string{
				"    test.txt:2:line2",
				"    test.txt:3:line3",
			},
		},
		{
			name:         "No match",
			input:        "hello world",
			regex:        "foo",
			contextLines: 1,
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := regexp.MustCompile(tt.regex)
			s := &Searcher{Regex: r, ContextLines: tt.contextLines}
			reader := strings.NewReader(tt.input)

			results := s.ScanStream(reader, "test.txt")

			if len(results) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(results))
				return
			}

			for i, res := range results {
				if res != tt.expected[i] {
					t.Errorf("result %d mismatch:\nexpected: %q\ngot:      %q", i, tt.expected[i], res)
				}
			}
		})
	}
}
