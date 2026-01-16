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

// ScanStream reads from a reader and calls the provided callback for each formatted result string.
func (s *Searcher) ScanStream(r io.Reader, filename string, onMatch func(string)) {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	contextLeft := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		isMatch := s.Regex.MatchString(line)
		if isMatch {
			contextLeft = s.ContextLines - 1
			onMatch(fmt.Sprintf("    %s:%d:%s", filename, lineNum, line))
		} else if contextLeft > 0 {
			onMatch(fmt.Sprintf("    %s:%d:%s", filename, lineNum, line))
			contextLeft--
		}
	}
}
