package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type HttpFragment struct {
	srcUrl    *url.URL
	startPos  int
	endPos    int
	resp      *http.Response
	semaphore chan struct{}
}

func (h *HttpFragment) Start(httpClient *http.Client, file *os.File) *bytes.Buffer {
	Println("Starting Fragment", h.startPos, " to ", h.endPos)
	// Check to see if we have response already
	if h.resp == nil {
		req := &http.Request{
			Method: "GET",
			URL:    h.srcUrl,
			Header: http.Header{
				"Range": {
					h.GetRange(),
				},
			},
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			Panic(err)
		}
		if !IsSuccessResp(resp) {
			Panic(fmt.Errorf("received non-200 code for fragment %v to %v: %v", h.startPos, h.endPos, resp.StatusCode))
		}
		h.resp = resp
	}
	defer h.resp.Body.Close()
	// Write contents to a buffer first
	Println("Allocating buffer", h.startPos, " to ", h.endPos)
	size := h.GetSize()
	buffer := bytes.NewBuffer(make([]byte, size))
	Println("Writing to buffer", h.startPos, " to ", h.endPos)
	_, err := io.Copy(buffer, h.resp.Body)
	if err != nil {
		Panic(err)
	}
	Println("Finished writing to buffer", h.startPos, " to ", h.endPos)
	<-h.semaphore
	return buffer
}

func (h *HttpFragment) GetSize() int {
	return min(h.endPos-h.startPos, MaxFragmentSize)
}

func (h *HttpFragment) GetRange() string {
	return fmt.Sprintf("bytes=%v-%v", h.startPos, h.endPos)
}
