package main

import (
	"errors"
	"io"
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
	writeMutex sync.Mutex
	semaphore  chan struct{}
}

func NewRequestManager(src string, dest *os.File) (*RequestManager, error) {
	srcUrl, err := url.ParseRequestURI(src)
	if err != nil {
		return nil, err
	}

	return &RequestManager{
		httpClient: http.DefaultClient,
		srcUrl:     srcUrl,
		destFile:   dest,
		wg:         sync.WaitGroup{},
		semaphore:  make(chan struct{}, Parallelization),
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

	numFragments := int(totalSize / MaxFragmentSize)
	if numFragments == 0 {
		numFragments = 1
	}
	Println("Num Fragments:", numFragments)
	for i := 0; i < numFragments; i++ {
		startPos := i * MaxFragmentSize
		endPos := min(startPos+MaxFragmentSize, int(totalSize))

		fragment := &HttpFragment{
			srcUrl:    r.srcUrl,
			startPos:  startPos,
			endPos:    endPos,
			semaphore: r.semaphore,
		}

		if i == 0 {
			fragment.resp = initResp
		}

		r.wg.Add(1)
		r.semaphore <- struct{}{}
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
	buffer := fragment.Start(r.httpClient, r.destFile)

	// Write the body to the file. Need a mutex first. Alternatively we can use channels for this purpose.
	Println("Seeking", fragment.startPos, " to ", fragment.endPos)
	_, err := r.destFile.Seek(int64(fragment.startPos), io.SeekStart)
	if err != nil {
		Panic(err)
	}
	Println("Writing", fragment.startPos, " to ", fragment.endPos)

	_, err = io.Copy(r.destFile, buffer)
	if err != nil {
		Panic(err)
	}
	Println("Finished writing", fragment.startPos, " to ", fragment.endPos)
}
