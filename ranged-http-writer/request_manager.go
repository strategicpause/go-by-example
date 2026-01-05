package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

// RequestManager orchestrates the entire download process
type RequestManager struct {
	httpClient    *http.Client   // Reusable HTTP client
	srcUrl        *url.URL       // Parsed source URL
	destFile      *os.File       // Destination file handle
	wg            sync.WaitGroup // Goroutine synchronization
	writeMutex    sync.Mutex     // File write synchronization
	semaphore     chan struct{}  // Concurrency limiting
	startTime     time.Time      // Download start time for logging
	totalSize     int64          // Total file size for progress tracking
	fragmentCount int            // Number of fragments
}

// NewRequestManager creates a new RequestManager instance
func NewRequestManager(src string, dest *os.File) (*RequestManager, error) {
	// Validate and parse URL
	if err := ValidateURL(src); err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	srcUrl, err := url.ParseRequestURI(src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	return &RequestManager{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Increase timeout for large downloads
		},
		srcUrl:    srcUrl,
		destFile:  dest,
		wg:        sync.WaitGroup{},
		semaphore: make(chan struct{}, Config.Parallelization),
		startTime: time.Now(),
	}, nil
}

// Start initiates the download process with optimized initial request
func (r *RequestManager) Start() error {
	r.startTime = time.Now()
	StartTime = r.startTime // Update global start time for logging

	// Perform initial request to determine file size
	initResp, err := r.initRequest()
	if err != nil {
		return fmt.Errorf("initial request failed: %v", err)
	}

	// Use the total size determined in initRequest, or parse from response if not set
	totalSize := r.totalSize
	if totalSize == 0 {
		// This happens when we fell back to a regular GET request
		contentLength := initResp.Header.Get("Content-Length")
		totalSize, err = strconv.ParseInt(contentLength, 10, 64)
		if err != nil || totalSize <= 0 {
			totalSize = 0
			Println("Content-Length not available, using single fragment download")
		}
		r.totalSize = totalSize
	}

	Println("Total file size:", totalSize, "bytes")

	// Calculate number of fragments using the new utility function
	r.fragmentCount = CalculateFragmentCount(totalSize, Config.MaxFragmentSize)
	Println("Number of fragments:", r.fragmentCount)

	// Calculate fragment ranges using the new utility function
	fragmentRanges := CalculateFragmentRanges(totalSize, Config.MaxFragmentSize)

	// Validate fragment ranges to ensure no gaps or overlaps
	if err := ValidateFragmentRanges(fragmentRanges); err != nil {
		return fmt.Errorf("fragment range validation failed: %v", err)
	}

	// Create and process fragments using the calculated ranges
	for i, fragmentRange := range fragmentRanges {
		fragment := NewHttpFragment(r.srcUrl, fragmentRange.StartPos, fragmentRange.EndPos)

		// Reuse initial response for first fragment (optimization)
		if i == 0 {
			fragment.resp = initResp
		}

		r.wg.Add(1)
		r.semaphore <- struct{}{} // Acquire semaphore
		go r.processFragment(fragment)
	}

	// Wait for all fragments to complete
	r.wg.Wait()

	Println("Download completed successfully")
	return nil
}

// initRequest performs initial request to determine file size and potentially download first fragment
func (r *RequestManager) initRequest() (*http.Response, error) {
	// First, make a HEAD request to get the total file size
	headReq := &http.Request{
		Method: "HEAD",
		URL:    r.srcUrl,
		Header: make(http.Header),
	}

	headResp, err := r.httpClient.Do(headReq)
	if err != nil {
		return nil, fmt.Errorf("HEAD request failed: %v", err)
	}
	defer headResp.Body.Close()

	if !IsSuccessResp(headResp) {
		return nil, fmt.Errorf("HEAD request returned non-2xx status code: %d", headResp.StatusCode)
	}

	// Get the total file size from the HEAD response
	contentLength := headResp.Header.Get("Content-Length")
	totalSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil || totalSize <= 0 {
		// If Content-Length is missing, fall back to a regular GET request
		Println("Content-Length not available from HEAD request, falling back to GET")
		req := &http.Request{
			Method: "GET",
			URL:    r.srcUrl,
			Header: make(http.Header),
		}

		resp, err := r.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GET request failed: %v", err)
		}

		if !IsSuccessResp(resp) {
			resp.Body.Close()
			return nil, fmt.Errorf("GET request returned non-2xx status code: %d", resp.StatusCode)
		}

		return resp, nil
	}

	// Store the total size for later use
	r.totalSize = totalSize

	// Now make a range request for the first fragment to optimize the download
	req := &http.Request{
		Method: "GET",
		URL:    r.srcUrl,
		Header: make(http.Header),
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", Config.MaxFragmentSize-1))

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Range request failed: %v", err)
	}

	if !IsSuccessResp(resp) {
		resp.Body.Close()
		return nil, fmt.Errorf("Range request returned non-2xx status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// IsSuccessResp checks if HTTP response has a 2xx status code
func IsSuccessResp(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// processFragment handles individual fragment processing with pipeline processing
func (r *RequestManager) processFragment(fragment *HttpFragment) {
	defer r.wg.Done()
	defer func() { <-r.semaphore }() // Release semaphore

	// Use pipeline processing - fragment writes directly to disk
	err := fragment.Start(r.httpClient, r.destFile, &r.writeMutex)
	if err != nil {
		Panic(fmt.Errorf("fragment processing failed: %v", err))
	}
}
