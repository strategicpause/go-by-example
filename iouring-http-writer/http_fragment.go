package main

import (
	"bytes"
	"fmt"
	"github.com/iceber/iouring-go"
	"io"
	"net/http"
	"net/url"
	"os"
)

const PageSize = 1 * 1024 * 1024

type HttpFragment struct {
	srcUrl   *url.URL
	startPos int
	endPos   int
	resp     *http.Response
}

func (h *HttpFragment) Start(httpClient *http.Client, file *os.File, iour *iouring.IOURing) {
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
	byteBuffer := make([]byte, PageSize)
	buffer := bytes.NewBuffer(byteBuffer)
	pos := uint64(h.startPos)
	resultCh := make(chan iouring.Result, 1)
	Println("Starting writes for", h.startPos, " to ", h.endPos)
	for {
		written, err := io.CopyN(buffer, h.resp.Body, PageSize)
		if err != nil && err != io.EOF {
			panic(err)
		}

		writeReq := iouring.Pwrite(int(file.Fd()), buffer.Bytes(), pos)
		if _, err := iour.SubmitRequest(writeReq, resultCh); err != nil {
			panic(err)
		}
		result := <-resultCh
		if err := result.Err(); err != nil {
			panic(err)
		}
		pos += uint64(written)
		buffer.Reset()
		if err == io.EOF || int(pos) >= h.endPos {
			break
		}
	}
}

func (h *HttpFragment) GetSize() int {
	return h.endPos - h.startPos
}

func (h *HttpFragment) GetRange() string {
	return fmt.Sprintf("bytes=%v-%v", h.startPos, h.endPos)
}
