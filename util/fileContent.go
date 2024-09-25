package util

import (
	"fmt"
	"regexp"
	"strings"
)

// CheckWordPresence checks if a specific word or phrase exists in the file content.
func CheckWordPresence(content, word string) bool {
	return strings.Contains(content, word)
}

// CheckRegexMatch checks if the content matches a specific regex pattern.
func CheckRegexMatch(content, regexPattern string) (bool, []string, error) {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false, nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	// Find all lines that match the regex pattern
	lines := strings.Split(content, "\n")
	var matchedLines []string
	for _, line := range lines {
		if re.MatchString(line) {
			matchedLines = append(matchedLines, line)
		}
	}

	return len(matchedLines) > 0, matchedLines, nil
}

// CheckLineCount validates if the content has the expected number of lines.
func CheckLineCount(content string, expectedLineCount int) bool {
	actualLineCount := len(strings.Split(content, "\n"))
	return actualLineCount == expectedLineCount
}
