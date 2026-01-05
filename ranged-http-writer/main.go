package main

import (
	"fmt"
	"os"
	"time"
)

// Configuration constants with safe defaults
const (
	MiB                    = 1024 * 1024
	DefaultMaxFragmentSize = 20 * MiB
	DefaultParallelization = 4
	DefaultDebug           = false
)

// Global configuration instance
var Config = NewConfiguration(DefaultMaxFragmentSize, DefaultParallelization, DefaultDebug)

// Progress tracking
var (
	StartTime = time.Now()
)

func main() {
	// Example usage - in real usage these would come from command line args
	targetUrl := "https://testfileorg.netwet.net/500MB-CZIPtestfile.org.zip"
	fileName := "writer.bin"

	// Validate URL
	if err := ValidateURL(targetUrl); err != nil {
		Panic(fmt.Errorf("invalid URL: %v", err))
	}

	// Create destination file
	file, err := os.Create(fileName)
	if err != nil {
		Panic(fmt.Errorf("failed to create file: %v", err))
	}
	defer file.Close()

	// Create and start request manager
	reqMgr, err := NewRequestManager(targetUrl, file)
	if err != nil {
		Panic(fmt.Errorf("failed to create request manager: %v", err))
	}

	err = reqMgr.Start()
	if err != nil {
		Panic(fmt.Errorf("download failed: %v", err))
	}
}
