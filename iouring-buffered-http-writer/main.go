package main

import (
	"fmt"
	"os"
	"time"
)

// TODO - Benchmark time & size with 1 & 10 GiB files.

const (
	Parallelization = 4
	TargetUrl       = "https://testfileorg.netwet.net/500MB-CZIPtestfile.org.zip"
	FileName        = "writer.bin"
	Debug           = true
)

var (
	Now = time.Now()
)

func Println(str ...any) {
	if Debug {
		now := time.Since(Now).Milliseconds()
		fmt.Println(now, "ms:", str)
	}
}

func Panic(str any) {
	now := time.Since(Now).Milliseconds()
	panic(fmt.Sprintf("%v ms: %s", now, str))
}

func main() {
	file, err := os.Create(FileName)
	defer file.Close()
	if err != nil {
		Panic(err)
	}

	reqMgr, err := NewRequestManager(TargetUrl, file)
	if err != nil {
		Panic(err)
	}
	err = reqMgr.Start()
	if err != nil {
		Panic(err)
	}
}
