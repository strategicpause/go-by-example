package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

const (
	TestRuns = 50
)

func TestSerial(t *testing.T) {
	ojp := NewOrderedJobProcessor(1)

	processed := testOrderedJobProcessor(ojp)

	assert.Equal(t, TestRuns, processed)
}

func testOrderedJobProcessor(ojp *OrderedJobProcessor) int {
	processed := 0

	for i := 0; i < TestRuns; i++ {
		job := func() error {
			r := time.Duration(rand.Intn(50))
			time.Sleep(r * time.Millisecond)

			return nil
		}
		cb := func() error {
			if processed != i {
				return fmt.Errorf("expected %d jobs, got %d", i, processed)
			}
			processed += 1

			return nil
		}
		ojp.SubmitJob(job, cb)
	}
	ojp.Wait()

	return processed
}

func TestParallel(t *testing.T) {
	ojp := NewOrderedJobProcessor(2)

	processed := testOrderedJobProcessor(ojp)

	assert.Equal(t, TestRuns, processed)
}

func BenchmarkSerial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ojp := NewOrderedJobProcessor(1)
		testOrderedJobProcessor(ojp)
	}
}

func BenchmarkParallel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ojp := NewOrderedJobProcessor(2)
		testOrderedJobProcessor(ojp)
	}
}
