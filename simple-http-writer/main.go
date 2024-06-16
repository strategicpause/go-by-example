package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	file, err := os.Create("writer.bin")
	defer file.Close()
	if err != nil {
		panic(err)
	}

	targetUrl, err := url.Parse("https://testfileorg.netwet.net/500MB-CZIPtestfile.org.zip")
	if err != nil {
		panic(err)
	}

	httpClient := http.DefaultClient

	req := &http.Request{
		Method: "GET",
		URL:    targetUrl,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}

	io.Copy(file, resp.Body)
}
