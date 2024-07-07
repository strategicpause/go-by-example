package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

const (
	defaultParallelization = 1
)

type Granger struct {
	httpClient      *http.Client
	srcUrl          *url.URL
	wg              sync.WaitGroup
	ojp             *OrderedJobProcessor
	fragmentSize    int
	parallelization int
}

type Option func(g *Granger)

func WithParallelization(parallelization int) Option {
	return func(g *Granger) {
		g.parallelization = parallelization
	}
}

func WithFragmentSize(fragmentSize int) Option {
	return func(g *Granger) {
		g.fragmentSize = fragmentSize
	}
}

func NewGranger(uri *url.URL, options ...Option) *Granger {
	g := &Granger{
		httpClient:      http.DefaultClient,
		srcUrl:          uri,
		wg:              sync.WaitGroup{},
		parallelization: defaultParallelization,
	}

	if uri.Scheme == "https" {
		g.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{},
		}
	}

	for _, opt := range options {
		opt(g)
	}

	ojp := NewOrderedJobProcessor(g.parallelization)
	g.ojp = ojp

	return g
}

func (r *Granger) WriteTo(w io.Writer) (int64, error) {
	initResp, err := r.initRequest()
	if err != nil {
		return 0, err
	}
	contentLength := initResp.Header.Get("Content-Length")
	// If we're unable to parse the content-length, then this will return an error and default to 0. Let's ignore
	// the error and use the default instead.
	totalSize, _ := strconv.ParseInt(contentLength, 10, 64)

	if r.fragmentSize == 0 {
		r.fragmentSize = int(totalSize)
	}
	numFragments := int(totalSize) / r.fragmentSize
	if numFragments == 0 {
		numFragments = 1
	}
	for i := 0; i < numFragments; i++ {
		startPos := i * r.fragmentSize
		endPos := min(startPos+r.fragmentSize, int(totalSize))

		fragment := &HttpFragment{
			srcUrl:   r.srcUrl,
			startPos: startPos,
			endPos:   endPos,
		}

		if i == 0 {
			fragment.resp = initResp
		}

		r.wg.Add(1)
		r.processFragment(fragment, w)
	}

	r.wg.Wait()

	return totalSize, nil
}

func (r *Granger) initRequest() (*http.Response, error) {
	req := &http.Request{
		Method:     "GET",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        r.srcUrl,
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if !isSuccessResp(resp) {
		return nil, errors.New("received non-200 code")
	}

	return resp, nil
}

func isSuccessResp(resp *http.Response) bool {
	return resp.StatusCode/100 == 2
}

func (r *Granger) processFragment(fragment *HttpFragment, w io.Writer) {
	var buff *bytes.Buffer
	job := func() error {
		buffer, err := fragment.Start(r.httpClient)
		if err != nil {
			return err
		}
		buff = buffer
		return nil
	}

	cb := func() error {
		defer r.wg.Done()
		_, err := io.Copy(w, buff)
		return err
	}

	r.ojp.SubmitJob(job, cb)
}
