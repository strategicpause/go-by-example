package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

const (
	Parallelization = 5
	MiB             = 1024 * 1024
	FragmentSize    = 20 * MiB
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("usage: %s URL.\n", os.Args[0])
	}

	uri, err := url.Parse(args[0])
	if err != nil {
		panic(err)
	}

	granger := NewGranger(
		uri,
		WithParallelization(Parallelization),
		WithFragmentSize(FragmentSize),
	)
	if _, err := granger.WriteTo(os.Stdout); err != nil {
		panic(err)
	}
}
