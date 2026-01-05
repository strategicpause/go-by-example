package main

import (
	"net/url"
	"os"
	"testing"
)

// TestBasicIntegration tests that the core components work together
func TestBasicIntegration(t *testing.T) {
	// Test that we can create a RequestManager with a valid URL
	tempFile, err := os.CreateTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	testURL := "https://httpbin.org/bytes/1024" // Small test file
	reqMgr, err := NewRequestManager(testURL, tempFile)
	if err != nil {
		t.Fatalf("Failed to create RequestManager: %v", err)
	}

	// Verify the URL was parsed correctly
	expectedURL, _ := url.ParseRequestURI(testURL)
	if reqMgr.srcUrl.String() != expectedURL.String() {
		t.Errorf("Expected URL %s, got %s", expectedURL.String(), reqMgr.srcUrl.String())
	}

	// Verify configuration is properly initialized
	if reqMgr.httpClient == nil {
		t.Error("HTTP client should be initialized")
	}

	if reqMgr.destFile != tempFile {
		t.Error("Destination file should match provided file")
	}

	if cap(reqMgr.semaphore) != Config.Parallelization {
		t.Errorf("Semaphore capacity should be %d, got %d", Config.Parallelization, cap(reqMgr.semaphore))
	}
}

// TestHttpFragmentCreation tests HttpFragment creation and basic methods
func TestHttpFragmentCreation(t *testing.T) {
	testURL, _ := url.ParseRequestURI("https://example.com/test.bin")

	fragment := NewHttpFragment(testURL, 0, 1023)

	if fragment.srcUrl != testURL {
		t.Error("Source URL should match provided URL")
	}

	if fragment.startPos != 0 {
		t.Error("Start position should be 0")
	}

	if fragment.endPos != 1023 {
		t.Error("End position should be 1023")
	}

	// Test range header formatting
	expectedRange := "bytes=0-1023"
	if fragment.GetRange() != expectedRange {
		t.Errorf("Expected range header %s, got %s", expectedRange, fragment.GetRange())
	}

	// Test size calculation
	expectedSize := 1024 // 1023 - 0 + 1
	if fragment.GetSize() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, fragment.GetSize())
	}
}

// TestConfigurationDefaults tests that configuration defaults work correctly
func TestConfigurationDefaults(t *testing.T) {
	// Test with invalid values
	config := NewConfiguration(-1, -1, true)

	if config.MaxFragmentSize != DefaultMaxFragmentSize {
		t.Errorf("Expected default fragment size %d, got %d", DefaultMaxFragmentSize, config.MaxFragmentSize)
	}

	if config.Parallelization != DefaultParallelization {
		t.Errorf("Expected default parallelization %d, got %d", DefaultParallelization, config.Parallelization)
	}

	if config.Debug != true {
		t.Error("Debug flag should be preserved when valid")
	}
}
