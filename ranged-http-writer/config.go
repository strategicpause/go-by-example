package main

import (
	"fmt"
)

// Validation limits
const (
	MinFragmentSize    = 1024      // 1 KiB minimum
	MaxFragmentSize    = 100 * MiB // 100 MiB maximum
	MinParallelization = 1         // At least 1 connection
	MaxParallelization = 32        // Maximum 32 concurrent connections
)

// Configuration holds system-wide configuration parameters
type Configuration struct {
	MaxFragmentSize int
	Parallelization int
	Debug           bool
}

// NewConfiguration creates a new configuration with validated parameters
func NewConfiguration(fragmentSize, parallelization int, debug bool) *Configuration {
	// Validate fragment size, use default if invalid
	validatedFragmentSize := fragmentSize
	if err := validateFragmentSize(fragmentSize); err != nil {
		validatedFragmentSize = DefaultMaxFragmentSize
	}

	// Validate parallelization factor, use default if invalid
	validatedParallelization := parallelization
	if err := validateParallelization(parallelization); err != nil {
		validatedParallelization = DefaultParallelization
	}

	// Debug flag validation - boolean values are always valid
	return &Configuration{
		MaxFragmentSize: validatedFragmentSize,
		Parallelization: validatedParallelization,
		Debug:           debug,
	}
}

// validateFragmentSize validates fragment size and returns an error if invalid
func validateFragmentSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("fragment size must be positive, got %d", size)
	}
	if size < MinFragmentSize {
		return fmt.Errorf("fragment size %d is below minimum %d", size, MinFragmentSize)
	}
	if size > MaxFragmentSize {
		return fmt.Errorf("fragment size %d exceeds maximum %d", size, MaxFragmentSize)
	}
	return nil
}

// validateParallelization validates parallelization factor and returns an error if invalid
func validateParallelization(factor int) error {
	if factor <= 0 {
		return fmt.Errorf("parallelization factor must be positive, got %d", factor)
	}
	if factor < MinParallelization {
		return fmt.Errorf("parallelization factor %d is below minimum %d", factor, MinParallelization)
	}
	if factor > MaxParallelization {
		return fmt.Errorf("parallelization factor %d exceeds maximum %d", factor, MaxParallelization)
	}
	return nil
}

// ValidateConfigurationValues validates configuration values and returns detailed errors
// This function can be used by external code to validate configuration before creating a Configuration
func ValidateConfigurationValues(fragmentSize, parallelization int, debug bool) error {
	if err := validateFragmentSize(fragmentSize); err != nil {
		return fmt.Errorf("invalid fragment size: %w", err)
	}
	if err := validateParallelization(parallelization); err != nil {
		return fmt.Errorf("invalid parallelization factor: %w", err)
	}
	// Debug flag is always valid (boolean)
	return nil
}
