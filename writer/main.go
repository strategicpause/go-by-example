package main

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

type Foo struct {
	io.WriterTo
}

func (f *Foo) WriteTo(w io.Writer) (n int64, err error) {
	msg := []byte("Hello World")
	w.Write(msg)
	return int64(len(msg)), nil
}

func main() {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	reader, writer := io.Pipe()
	defer writer.Close()

	foo := Foo{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer := &bytes.Buffer{}
		io.Copy(buffer, reader)
		fmt.Println(string(buffer.Bytes()))
	}()

	foo.WriteTo(writer)
}
