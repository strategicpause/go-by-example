package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Println logs debug messages with timestamps when debug mode is enabled
func Println(args ...any) {
	if Config.Debug {
		elapsed := time.Since(StartTime).Milliseconds()
		fmt.Printf("%d ms: ", elapsed)
		fmt.Println(args...)
	}
}

// Panic logs error with timestamp and panics
func Panic(err any) {
	elapsed := time.Since(StartTime).Milliseconds()
	panic(fmt.Sprintf("%d ms: %v", elapsed, err))
}

// ValidateURL validates that the provided URL is a proper HTTP or HTTPS URL
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Trim whitespace
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty or whitespace only")
	}

	// Parse the URL using Go's standard library
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("malformed URL: %v", err)
	}

	// Validate scheme - must be HTTP or HTTPS
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Validate host - must be present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}

// FragmentRange represents a byte range for a fragment
type FragmentRange struct {
	StartPos int // Starting byte position (inclusive)
	EndPos   int // Ending byte position (inclusive)
}

// CalculateFragmentCount calculates the number of fragments needed for a given file size
// Returns 1 for small files or when file size is unknown (0)
func CalculateFragmentCount(fileSize int64, maxFragmentSize int) int {
	// Handle edge cases for small files and unknown size
	if fileSize <= 0 || fileSize <= int64(maxFragmentSize) {
		return 1
	}

	// Calculate number of fragments using ceiling division
	// This ensures we have enough fragments to cover the entire file
	fragmentCount := int((fileSize + int64(maxFragmentSize) - 1) / int64(maxFragmentSize))

	// Ensure we always have at least 1 fragment
	if fragmentCount < 1 {
		return 1
	}

	return fragmentCount
}

// CalculateFragmentRanges calculates byte ranges for all fragments
// Ensures contiguous ranges with no gaps or overlaps
func CalculateFragmentRanges(fileSize int64, maxFragmentSize int) []FragmentRange {
	fragmentCount := CalculateFragmentCount(fileSize, maxFragmentSize)
	ranges := make([]FragmentRange, fragmentCount)

	for i := 0; i < fragmentCount; i++ {
		startPos := i * maxFragmentSize
		var endPos int

		if fileSize > 0 {
			// For known file size, ensure we don't exceed the file boundary
			endPos = min(startPos+maxFragmentSize-1, int(fileSize)-1)
		} else {
			// For unknown file size, use standard fragment size
			endPos = startPos + maxFragmentSize - 1
		}

		ranges[i] = FragmentRange{
			StartPos: startPos,
			EndPos:   endPos,
		}
	}

	return ranges
}

// ValidateFragmentRanges validates that fragment ranges are contiguous with no gaps or overlaps
func ValidateFragmentRanges(ranges []FragmentRange) error {
	if len(ranges) == 0 {
		return fmt.Errorf("no fragment ranges provided")
	}

	// Single fragment is always valid
	if len(ranges) == 1 {
		if ranges[0].StartPos < 0 {
			return fmt.Errorf("fragment start position cannot be negative: %d", ranges[0].StartPos)
		}
		if ranges[0].EndPos < ranges[0].StartPos {
			return fmt.Errorf("fragment end position (%d) cannot be less than start position (%d)", ranges[0].EndPos, ranges[0].StartPos)
		}
		return nil
	}

	// Validate multiple fragments
	for i := 0; i < len(ranges); i++ {
		currentRange := ranges[i]

		// Validate individual range
		if currentRange.StartPos < 0 {
			return fmt.Errorf("fragment %d start position cannot be negative: %d", i, currentRange.StartPos)
		}
		if currentRange.EndPos < currentRange.StartPos {
			return fmt.Errorf("fragment %d end position (%d) cannot be less than start position (%d)", i, currentRange.EndPos, currentRange.StartPos)
		}

		// Validate continuity with next fragment
		if i < len(ranges)-1 {
			nextRange := ranges[i+1]

			// Check for gaps: next fragment should start exactly where current fragment ends + 1
			if nextRange.StartPos != currentRange.EndPos+1 {
				return fmt.Errorf("gap detected between fragment %d (ends at %d) and fragment %d (starts at %d)", i, currentRange.EndPos, i+1, nextRange.StartPos)
			}
		}
	}

	return nil
}
