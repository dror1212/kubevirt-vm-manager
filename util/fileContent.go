package util

import (
	"regexp"
	"strings"
)

// CheckWordPresence checks if a specific word or phrase exists in the file content.
func CheckWordPresence(content, word string) bool {
	LogInfo("Checking for presence of word '%s' in content.", word)
	return strings.Contains(content, word)
}

// CheckRegexMatch checks if the content matches a specific regex pattern.
func CheckRegexMatch(content, regexPattern string) (bool, []string, error) {
	LogInfo("Checking content for regex pattern: %s", regexPattern)
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		LogError("Invalid regex pattern: %v", err)
		return false, nil, err
	}

	// Find all lines that match the regex pattern
	lines := strings.Split(content, "\n")
	var matchedLines []string
	for _, line := range lines {
		if re.MatchString(line) {
			matchedLines = append(matchedLines, line)
		}
	}

	LogInfo("Found %d lines matching the regex pattern.", len(matchedLines))
	return len(matchedLines) > 0, matchedLines, nil
}

// CheckLineCount validates if the content has the expected number of lines.
func CheckLineCount(content string, expectedLineCount int) bool {
	actualLineCount := len(strings.Split(content, "\n"))
	LogInfo("Checking line count. Expected: %d, Actual: %d", expectedLineCount, actualLineCount)
	return actualLineCount == expectedLineCount
}
