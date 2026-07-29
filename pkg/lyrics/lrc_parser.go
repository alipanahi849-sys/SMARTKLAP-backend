package lyrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LyricLine represents a single lyric line with timestamp
type LyricLine struct {
	TimestampMs int64  `json:"timestamp_ms"`
	Text        string `json:"text"`
}

// ParseLRC parses LRC format lyrics and returns structured lyric lines
// LRC format example: [00:10.500] First line
func ParseLRC(content string) ([]LyricLine, error) {
	var lines []LyricLine

	// Split content into lines
	contentLines := strings.Split(content, "\n")

	// Regex to match LRC timestamp format: [mm:ss.xx] or [mm:ss.xxx]
	lrcRegex := regexp.MustCompile(`^\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)$`)

	for _, line := range contentLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := lrcRegex.FindStringSubmatch(line)
		if matches == nil {
			// If line doesn't match LRC format, treat as plain text with no timestamp
			lines = append(lines, LyricLine{
				TimestampMs: 0,
				Text:        line,
			})
			continue
		}

		// Parse minutes, seconds, and milliseconds
		minutes, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minutes in timestamp: %w", err)
		}

		seconds, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("invalid seconds in timestamp: %w", err)
		}

		milliseconds, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid milliseconds in timestamp: %w", err)
		}

		// Adjust milliseconds if it's 2 digits (e.g., 50 -> 500)
		if len(matches[3]) == 2 {
			milliseconds *= 10
		}

		// Calculate total timestamp in milliseconds
		timestampMs := int64(minutes*60*1000 + seconds*1000 + milliseconds)

		// Extract lyric text
		text := strings.TrimSpace(matches[4])

		lines = append(lines, LyricLine{
			TimestampMs: timestampMs,
			Text:        text,
		})
	}

	return lines, nil
}

// ParsePlainText parses plain text lyrics (one line per lyric)
func ParsePlainText(content string) []LyricLine {
	var lines []LyricLine

	contentLines := strings.Split(content, "\n")
	for _, line := range contentLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lines = append(lines, LyricLine{
			TimestampMs: 0,
			Text:        line,
		})
	}

	return lines
}

// DetectFormat detects whether the content is LRC format or plain text
func DetectFormat(content string) string {
	// Check if any line matches LRC format
	lrcRegex := regexp.MustCompile(`^\[\d{2}:\d{2}\.\d{2,3}\]`)

	contentLines := strings.Split(content, "\n")
	for _, line := range contentLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if lrcRegex.MatchString(line) {
			return "lrc"
		}
	}

	return "plain"
}
