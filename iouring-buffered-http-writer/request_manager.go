package main

import (
	"errors"
	"github.com/iceber/iouring-go"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
)

type RequestManager struct {
	httpClient *http.Client
	srcUrl     *url.URL
	destFile   *os.File
	wg         sync.WaitGroup
	iour       *iouring.IOURing
}

func NewRequestManager(src string, dest *os.File) (*RequestManager, error) {
	srcUrl, err := url.ParseRequestURI(src)
	if err != nil {
		return nil, err
	}

	iour, err := iouring.New(1)
	if err != nil {
		return nil, err
	}

	return &RequestManager{
		httpClient: http.DefaultClient,
		srcUrl:     srcUrl,
		destFile:   dest,
		iour:       iour,
	}, nil
}

func (r *RequestManager) Start() error {
	initResp, err := r.initRequest()
	if err != nil {
		return err
	}
	contentLength := initResp.Header.Get("Content-Length")
	// If we're unable to parse the content-length, then this will return an error and default to 0. Let's ignore
	// the error and use the default instead.
	totalSize, _ := strconv.ParseInt(contentLength, 10, 64)
	Println("Size:", totalSize)

	fragmentSize := int(totalSize / Parallelization)
	for i := 0; i < Parallelization; i++ {
		startPos := i * fragmentSize
		endPos := min(startPos+fragmentSize, int(totalSize)) - 1

		fragment := &HttpFragment{
			srcUrl:   r.srcUrl,
			startPos: startPos,
			endPos:   endPos,
		}

		if i == 0 {
			fragment.resp = initResp
		}

		r.wg.Add(1)
		go r.processFragment(fragment)
	}

	r.wg.Wait()

	return nil
}

func (r *RequestManager) initRequest() (*http.Response, error) {
	req := &http.Request{
		Method: "GET",
		URL:    r.srcUrl,
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if !IsSuccessResp(resp) {
		return nil, errors.New("received non-200 code")
	}

	return resp, nil
}

func IsSuccessResp(resp *http.Response) bool {
	return resp.StatusCode/100 == 2
}

func (r *RequestManager) processFragment(fragment *HttpFragment) {
	defer r.wg.Done()
	Println("Writing", fragment.startPos, " to ", fragment.endPos)
	bytesCh := fragment.Start(r.httpClient)

	pos := uint64(fragment.startPos)
	resultCh := make(chan iouring.Result, 1)
	for buffer := range bytesCh {
		writeReq := iouring.Pwrite(int(r.destFile.Fd()), buffer.Bytes(), pos)
		if _, err := r.iour.SubmitRequest(writeReq, resultCh); err != nil {
			panic(err)
		}
		result := <-resultCh
		if err := result.Err(); err != nil {
			panic(err)
		}
		pos += uint64(buffer.Len())
	}

	Println("Finished writing", fragment.startPos, " to ", fragment.endPos)
}
