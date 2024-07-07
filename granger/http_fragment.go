package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type HttpFragment struct {
	srcUrl   *url.URL
	startPos int
	endPos   int
	resp     *http.Response
}

func (h *HttpFragment) Start(httpClient *http.Client) (*bytes.Buffer, error) {
	// Check to see if we have response already
	if h.resp == nil {
		req := &http.Request{
			Method: "GET",
			URL:    h.srcUrl,
			Header: http.Header{
				"Range": {
					fmt.Sprintf("bytes=%v-%v", h.startPos, h.endPos),
				},
			},
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if !isSuccessResp(resp) {
			return nil, fmt.Errorf("received non-200 code: %v", resp.StatusCode)
		}
		h.resp = resp
	}
	defer h.resp.Body.Close()
	size := h.endPos - h.startPos
	buffer := &bytes.Buffer{}
	_, err := io.CopyN(buffer, h.resp.Body, int64(size))
	if err != nil {
		return nil, err
	}
	return buffer, nil
}
