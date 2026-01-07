package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

// MatchResult represents a single match found in a file
type MatchResult struct {
	Filename string
	LineNum  int
	Content  string
}

// Searcher handles the scanning of content for matches
type Searcher struct {
	Regex        *regexp.Regexp
	ContextLines int
}

// ScanStream reads from a reader and returns a slice of formatted result strings.
// Using a callback or channel would be better for memory/streaming, but returning a slice is simple for now.
func (s *Searcher) ScanStream(r io.Reader, filename string) []string {
	scanner := bufio.NewScanner(r)
	var results []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if s.Regex.MatchString(line) {
			// Found a match
			results = append(results, fmt.Sprintf("    %s:%d:%s", filename, lineNum, line))

			// Handle context lines
			remaining := s.ContextLines - 1
			for remaining > 0 && scanner.Scan() {
				lineNum++
				nextLine := scanner.Text()
				results = append(results, fmt.Sprintf("    %s:%d:%s", filename, lineNum, nextLine))
				remaining--
			}
		}
	}
	return results
}
