package main

import (
	"testing"
	"testing/quick"
)

// Test configuration validation with property-based testing
func TestConfigurationValidation(t *testing.T) {
	// Test that invalid fragment sizes default to safe values
	f := func(fragmentSize, parallelization int, debug bool) bool {
		config := NewConfiguration(fragmentSize, parallelization, debug)

		// Fragment size should be within valid range or default to DefaultMaxFragmentSize
		if fragmentSize <= 0 || fragmentSize < MinFragmentSize || fragmentSize > MaxFragmentSize {
			return config.MaxFragmentSize == DefaultMaxFragmentSize
		} else {
			return config.MaxFragmentSize == fragmentSize
		}
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Configuration fragment size validation failed: %v", err)
	}
}

func TestParallelizationValidation(t *testing.T) {
	// Test that invalid parallelization factors default to safe values
	f := func(fragmentSize, parallelization int, debug bool) bool {
		config := NewConfiguration(fragmentSize, parallelization, debug)

		// Parallelization should be within valid range or default to DefaultParallelization
		if parallelization <= 0 || parallelization < MinParallelization || parallelization > MaxParallelization {
			return config.Parallelization == DefaultParallelization
		} else {
			return config.Parallelization == parallelization
		}
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Configuration parallelization validation failed: %v", err)
	}
}

// Test debug flag handling
func TestDebugFlagValidation(t *testing.T) {
	// Test that debug flag is properly set regardless of other parameters
	f := func(fragmentSize, parallelization int, debug bool) bool {
		config := NewConfiguration(fragmentSize, parallelization, debug)
		// Debug flag should always match the input (boolean values are always valid)
		return config.Debug == debug
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Configuration debug flag validation failed: %v", err)
	}
}

// Test edge cases for configuration validation
func TestConfigurationEdgeCases(t *testing.T) {
	// Test minimum valid values
	config := NewConfiguration(MinFragmentSize, MinParallelization, true)
	if config.MaxFragmentSize != MinFragmentSize {
		t.Errorf("Expected fragment size %d, got %d", MinFragmentSize, config.MaxFragmentSize)
	}
	if config.Parallelization != MinParallelization {
		t.Errorf("Expected parallelization %d, got %d", MinParallelization, config.Parallelization)
	}
	if !config.Debug {
		t.Errorf("Expected debug to be true")
	}

	// Test maximum valid values
	config = NewConfiguration(MaxFragmentSize, MaxParallelization, false)
	if config.MaxFragmentSize != MaxFragmentSize {
		t.Errorf("Expected fragment size %d, got %d", MaxFragmentSize, config.MaxFragmentSize)
	}
	if config.Parallelization != MaxParallelization {
		t.Errorf("Expected parallelization %d, got %d", MaxParallelization, config.Parallelization)
	}
	if config.Debug {
		t.Errorf("Expected debug to be false")
	}

	// Test values just outside valid ranges
	config = NewConfiguration(MinFragmentSize-1, MinParallelization-1, true)
	if config.MaxFragmentSize != DefaultMaxFragmentSize {
		t.Errorf("Expected default fragment size %d, got %d", DefaultMaxFragmentSize, config.MaxFragmentSize)
	}
	if config.Parallelization != DefaultParallelization {
		t.Errorf("Expected default parallelization %d, got %d", DefaultParallelization, config.Parallelization)
	}

	config = NewConfiguration(MaxFragmentSize+1, MaxParallelization+1, false)
	if config.MaxFragmentSize != DefaultMaxFragmentSize {
		t.Errorf("Expected default fragment size %d, got %d", DefaultMaxFragmentSize, config.MaxFragmentSize)
	}
	if config.Parallelization != DefaultParallelization {
		t.Errorf("Expected default parallelization %d, got %d", DefaultParallelization, config.Parallelization)
	}
}

// Test validation functions return proper errors
func TestValidationErrors(t *testing.T) {
	// Test fragment size validation errors
	testCases := []struct {
		size        int
		shouldError bool
		description string
	}{
		{0, true, "zero fragment size"},
		{-1, true, "negative fragment size"},
		{MinFragmentSize - 1, true, "below minimum fragment size"},
		{MaxFragmentSize + 1, true, "above maximum fragment size"},
		{MinFragmentSize, false, "minimum valid fragment size"},
		{MaxFragmentSize, false, "maximum valid fragment size"},
		{DefaultMaxFragmentSize, false, "default fragment size"},
	}

	for _, tc := range testCases {
		err := validateFragmentSize(tc.size)
		if tc.shouldError && err == nil {
			t.Errorf("Expected error for %s (size: %d), but got none", tc.description, tc.size)
		}
		if !tc.shouldError && err != nil {
			t.Errorf("Expected no error for %s (size: %d), but got: %v", tc.description, tc.size, err)
		}
	}

	// Test parallelization validation errors
	parallelizationCases := []struct {
		factor      int
		shouldError bool
		description string
	}{
		{0, true, "zero parallelization factor"},
		{-1, true, "negative parallelization factor"},
		{MinParallelization - 1, true, "below minimum parallelization factor"},
		{MaxParallelization + 1, true, "above maximum parallelization factor"},
		{MinParallelization, false, "minimum valid parallelization factor"},
		{MaxParallelization, false, "maximum valid parallelization factor"},
		{DefaultParallelization, false, "default parallelization factor"},
	}

	for _, tc := range parallelizationCases {
		err := validateParallelization(tc.factor)
		if tc.shouldError && err == nil {
			t.Errorf("Expected error for %s (factor: %d), but got none", tc.description, tc.factor)
		}
		if !tc.shouldError && err != nil {
			t.Errorf("Expected no error for %s (factor: %d), but got: %v", tc.description, tc.factor, err)
		}
	}
}

// Test the public ValidateConfigurationValues function
func TestValidateConfigurationValues(t *testing.T) {
	// Test valid configuration
	err := ValidateConfigurationValues(DefaultMaxFragmentSize, DefaultParallelization, false)
	if err != nil {
		t.Errorf("Expected no error for valid configuration, got: %v", err)
	}

	// Test invalid fragment size
	err = ValidateConfigurationValues(0, DefaultParallelization, false)
	if err == nil {
		t.Errorf("Expected error for invalid fragment size, got none")
	}

	// Test invalid parallelization
	err = ValidateConfigurationValues(DefaultMaxFragmentSize, 0, false)
	if err == nil {
		t.Errorf("Expected error for invalid parallelization, got none")
	}

	// Test both invalid
	err = ValidateConfigurationValues(0, 0, false)
	if err == nil {
		t.Errorf("Expected error for both invalid values, got none")
	}
}
