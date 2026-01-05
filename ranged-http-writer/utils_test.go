package main

import (
	"strings"
	"testing"
)

// Test URL validation
func TestURLValidation(t *testing.T) {
	// Test valid URLs
	validUrls := []string{
		"http://example.com",
		"https://example.com",
		"http://example.com/path",
		"https://example.com/path/to/file.zip",
		"http://subdomain.example.com",
		"https://example.com:8080/path",
		"http://192.168.1.1/file.txt",
		"https://example.com/path?query=value",
		"http://example.com/path#fragment",
		"https://example.com/path?query=value&other=test#fragment",
	}

	for _, url := range validUrls {
		if err := ValidateURL(url); err != nil {
			t.Errorf("Valid URL %s was rejected: %v", url, err)
		}
	}

	// Test invalid URLs
	invalidUrls := []string{
		"",                        // Empty string
		"   ",                     // Whitespace only
		"ftp://example.com",       // Wrong scheme
		"example.com",             // Missing scheme
		"http://",                 // Missing host
		"https://",                // Missing host
		"http",                    // Incomplete
		"https",                   // Incomplete
		"not-a-url",               // Not a URL
		"://example.com",          // Missing scheme
		"http:///path",            // Missing host
		"https:///path",           // Missing host
		"file://example.com",      // Wrong scheme
		"mailto:user@example.com", // Wrong scheme
		"http://",                 // No host after scheme
		"https://",                // No host after scheme
		"http:// example.com",     // Space in URL
		"https://exam ple.com",    // Space in host
	}

	for _, url := range invalidUrls {
		if err := ValidateURL(url); err == nil {
			t.Errorf("Invalid URL %s was accepted", url)
		}
	}
}

// TestCalculateFragmentCount tests the fragment count calculation function
func TestCalculateFragmentCount(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		maxFragmentSize int
		expected        int
	}{
		{"Small file", 1024, 20 * MiB, 1},
		{"Exact fragment size", int64(20 * MiB), 20 * MiB, 1},
		{"Slightly larger than fragment size", int64(20*MiB + 1), 20 * MiB, 2},
		{"Multiple fragments", int64(50 * MiB), 20 * MiB, 3},
		{"Unknown file size (zero)", 0, 20 * MiB, 1},
		{"Negative file size", -1, 20 * MiB, 1},
		{"Large file", int64(100 * MiB), 20 * MiB, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFragmentCount(tt.fileSize, tt.maxFragmentSize)
			if result != tt.expected {
				t.Errorf("CalculateFragmentCount(%d, %d) = %d, want %d", tt.fileSize, tt.maxFragmentSize, result, tt.expected)
			}
		})
	}
}

// TestCalculateFragmentRanges tests the fragment range calculation function
func TestCalculateFragmentRanges(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		maxFragmentSize int
		expectedRanges  []FragmentRange
	}{
		{
			"Small file",
			1024,
			20 * MiB,
			[]FragmentRange{{StartPos: 0, EndPos: 1023}},
		},
		{
			"Exact fragment size",
			int64(20 * MiB),
			20 * MiB,
			[]FragmentRange{{StartPos: 0, EndPos: 20*MiB - 1}},
		},
		{
			"Two fragments",
			int64(30 * MiB),
			20 * MiB,
			[]FragmentRange{
				{StartPos: 0, EndPos: 20*MiB - 1},
				{StartPos: 20 * MiB, EndPos: 30*MiB - 1},
			},
		},
		{
			"Unknown file size",
			0,
			20 * MiB,
			[]FragmentRange{{StartPos: 0, EndPos: 20*MiB - 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFragmentRanges(tt.fileSize, tt.maxFragmentSize)
			if len(result) != len(tt.expectedRanges) {
				t.Errorf("CalculateFragmentRanges(%d, %d) returned %d ranges, want %d", tt.fileSize, tt.maxFragmentSize, len(result), len(tt.expectedRanges))
				return
			}

			for i, expectedRange := range tt.expectedRanges {
				if result[i].StartPos != expectedRange.StartPos || result[i].EndPos != expectedRange.EndPos {
					t.Errorf("CalculateFragmentRanges(%d, %d)[%d] = {%d, %d}, want {%d, %d}",
						tt.fileSize, tt.maxFragmentSize, i,
						result[i].StartPos, result[i].EndPos,
						expectedRange.StartPos, expectedRange.EndPos)
				}
			}
		})
	}
}

// TestValidateFragmentRanges tests the fragment range validation function
func TestValidateFragmentRanges(t *testing.T) {
	tests := []struct {
		name        string
		ranges      []FragmentRange
		expectError bool
		errorMsg    string
	}{
		{
			"Valid single fragment",
			[]FragmentRange{{StartPos: 0, EndPos: 1023}},
			false,
			"",
		},
		{
			"Valid multiple fragments",
			[]FragmentRange{
				{StartPos: 0, EndPos: 1023},
				{StartPos: 1024, EndPos: 2047},
			},
			false,
			"",
		},
		{
			"Empty ranges",
			[]FragmentRange{},
			true,
			"no fragment ranges provided",
		},
		{
			"Negative start position",
			[]FragmentRange{{StartPos: -1, EndPos: 1023}},
			true,
			"fragment start position cannot be negative",
		},
		{
			"End position less than start position",
			[]FragmentRange{{StartPos: 1024, EndPos: 1023}},
			true,
			"fragment end position (1023) cannot be less than start position (1024)",
		},
		{
			"Gap between fragments",
			[]FragmentRange{
				{StartPos: 0, EndPos: 1023},
				{StartPos: 1025, EndPos: 2047}, // Gap at 1024
			},
			true,
			"gap detected between fragment 0 (ends at 1023) and fragment 1 (starts at 1025)",
		},
		{
			"Overlapping fragments",
			[]FragmentRange{
				{StartPos: 0, EndPos: 1024},
				{StartPos: 1024, EndPos: 2047}, // Overlap at 1024
			},
			true,
			"gap detected between fragment 0 (ends at 1024) and fragment 1 (starts at 1024)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFragmentRanges(tt.ranges)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateFragmentRanges() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateFragmentRanges() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFragmentRanges() unexpected error = %v", err)
				}
			}
		})
	}
}
