package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHappyCase(t *testing.T) {
	payload := []byte("hello world")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(payload)
		assert.NoError(t, err)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	assert.NoError(t, err)

	g := NewGranger(u)
	buffer := &bytes.Buffer{}

	_, err = g.WriteTo(buffer)
	assert.NoError(t, err)

	assert.Equal(t, buffer.Bytes(), payload)
}

func BenchmarkHappyCase(b *testing.B) {
	payload := []byte("hello world")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	buffer := &bytes.Buffer{}

	for i := 0; i < b.N; i++ {
		g := NewGranger(u)
		buffer.Reset()
		_, _ = g.WriteTo(buffer)
	}
}
