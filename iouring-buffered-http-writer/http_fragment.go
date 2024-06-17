package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const PageSize = 1 * 1024 * 1024
const Buffer = 5

type HttpFragment struct {
	srcUrl   *url.URL
	startPos int
	endPos   int
	resp     *http.Response
}

func (h *HttpFragment) Start(httpClient *http.Client) chan *bytes.Buffer {
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

	bytesCh := make(chan *bytes.Buffer, Buffer)

	go func() {
		Println("Starting writes for", h.startPos, " to ", h.endPos)
		size := h.GetSize()
		pos := 0
		defer h.resp.Body.Close()
		for {
			byteBuffer := make([]byte, PageSize)
			buffer := bytes.NewBuffer(byteBuffer)
			written, err := io.CopyN(buffer, h.resp.Body, PageSize)
			if err != nil && err != io.EOF {
				panic(err)
			}
			bytesCh <- buffer
			pos += int(written)
			if err == io.EOF || pos == size {
				fmt.Println("Pos:", pos, "Len:", buffer.Len(), "Written:", written, "Err:", err)
				break
			}
		}
		close(bytesCh)
	}()

	return bytesCh
}

func (h *HttpFragment) GetSize() int {
	return h.endPos - h.startPos
}

func (h *HttpFragment) GetRange() string {
	return fmt.Sprintf("bytes=%v-%v", h.startPos, h.endPos)
}
