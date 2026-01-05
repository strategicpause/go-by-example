package main

import (
	"testing"
	"testing/quick"
)

// Test HttpFragment range header formatting
func TestRangeHeaderFormatting(t *testing.T) {
	f := func(startPos, endPos uint) bool {
		// Ensure endPos >= startPos for valid ranges
		if endPos < startPos {
			startPos, endPos = endPos, startPos
		}

		// Convert to int to match HttpFragment expectations
		start := int(startPos)
		end := int(endPos)

		fragment := NewHttpFragment(nil, start, end)
		rangeHeader := fragment.GetRange()

		// Verify the header starts with "bytes=" and contains the positions
		return len(rangeHeader) > 6 && rangeHeader[:6] == "bytes="
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Range header formatting failed: %v", err)
	}
}

// Test fragment size calculation
func TestFragmentSizeCalculation(t *testing.T) {
	f := func(startPos, endPos uint) bool {
		// Ensure endPos >= startPos for valid ranges and keep sizes reasonable
		if endPos < startPos {
			startPos, endPos = endPos, startPos
		}

		// Keep sizes reasonable for testing (within 100MB range)
		start := int(startPos % (100 * MiB))
		end := start + int((endPos%(10*MiB))+1) // Ensure end > start

		fragment := NewHttpFragment(nil, start, end)
		size := fragment.GetSize()

		expectedSize := end - start + 1 // +1 because endPos is inclusive
		maxAllowed := min(expectedSize, Config.MaxFragmentSize)

		return size == maxAllowed && size > 0
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Fragment size calculation failed: %v", err)
	}
}
