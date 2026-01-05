package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
)

// HttpFragment represents a single chunk of a file to be downloaded via HTTP range request
type HttpFragment struct {
	srcUrl   *url.URL       // Source URL for the download
	startPos int            // Starting byte position (inclusive)
	endPos   int            // Ending byte position (inclusive)
	resp     *http.Response // Cached HTTP response (optional)
}

// NewHttpFragment creates a new HttpFragment with the specified parameters
func NewHttpFragment(srcUrl *url.URL, startPos, endPos int) *HttpFragment {
	return &HttpFragment{
		srcUrl:   srcUrl,
		startPos: startPos,
		endPos:   endPos,
		resp:     nil,
	}
}

// Start downloads the fragment data using pipeline processing and writes directly to disk
func (h *HttpFragment) Start(httpClient *http.Client, destFile *os.File, writeMutex *sync.Mutex) error {
	Println("Starting Fragment", h.startPos, "to", h.endPos)

	// Check if we have response already (optimization for first fragment)
	if h.resp == nil {
		req := &http.Request{
			Method: "GET",
			URL:    h.srcUrl,
			Header: http.Header{
				"Range": {h.GetRange()},
			},
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed for fragment %d-%d: %v", h.startPos, h.endPos, err)
		}

		if !IsSuccessResp(resp) {
			resp.Body.Close()
			return fmt.Errorf("received non-2xx status code for fragment %d-%d: %d", h.startPos, h.endPos, resp.StatusCode)
		}

		h.resp = resp
	}
	defer h.resp.Body.Close()

	// Pipeline processing: stream data in 64 KiB chunks directly to disk
	const chunkSize = 64 * 1024 // 64 KiB chunks
	buffer := make([]byte, chunkSize)
	currentPos := h.startPos
	totalWritten := int64(0)

	Println("Starting pipeline processing for fragment", h.startPos, "to", h.endPos)

	for {
		// Read chunk from HTTP response
		n, err := h.resp.Body.Read(buffer)
		if n > 0 {
			// Calculate how much we should actually write (don't exceed fragment end)
			remainingBytes := h.endPos - currentPos + 1
			bytesToWrite := min(n, remainingBytes)

			if bytesToWrite > 0 {
				// Write chunk directly to disk with proper synchronization
				writeMutex.Lock()

				// Seek to current position
				_, seekErr := destFile.Seek(int64(currentPos), io.SeekStart)
				if seekErr != nil {
					writeMutex.Unlock()
					return fmt.Errorf("failed to seek to position %d for fragment %d-%d: %v", currentPos, h.startPos, h.endPos, seekErr)
				}

				// Write the chunk
				written, writeErr := destFile.Write(buffer[:bytesToWrite])
				writeMutex.Unlock()

				if writeErr != nil {
					return fmt.Errorf("failed to write chunk for fragment %d-%d: %v", h.startPos, h.endPos, writeErr)
				}

				currentPos += written
				totalWritten += int64(written)

				// Check if we've written all the data for this fragment
				if currentPos > h.endPos {
					break
				}
			}
		}

		// Check for read errors
		if err != nil {
			if err == io.EOF {
				break // End of response
			}
			return fmt.Errorf("failed to read chunk for fragment %d-%d: %v", h.startPos, h.endPos, err)
		}

		// Safety check to prevent infinite loops
		if currentPos > h.endPos {
			break
		}
	}

	Println("Completed pipeline processing for fragment", h.startPos, "to", h.endPos, "- wrote", totalWritten, "bytes")
	return nil
}

// GetSize calculates the actual fragment size
func (h *HttpFragment) GetSize() int {
	actualSize := h.endPos - h.startPos + 1 // +1 because endPos is inclusive
	return min(actualSize, Config.MaxFragmentSize)
}

// GetRange formats the HTTP Range header
func (h *HttpFragment) GetRange() string {
	return fmt.Sprintf("bytes=%d-%d", h.startPos, h.endPos)
}
