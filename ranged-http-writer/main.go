package main

import (
	"fmt"
	"os"
	"time"
)

const (
	MiB             = 1024 * 1024
	MaxFragmentSize = 20 * MiB
	Parallelization = 4
	TargetUrl       = "https://testfileorg.netwet.net/500MB-CZIPtestfile.org.zip"
	FileName        = "writer.bin"
	Debug           = false
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
