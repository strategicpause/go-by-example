package main

import (
	"fmt"
	"math/rand"
	"sync"
)

const (
	Parallelization = 4
	MaxExecutions   = 100
)

var (
	Semaphore      = make(chan struct{}, Parallelization)
	CompletedExecs = sync.Map{}
	Callback       = make(chan struct{})
	Current        = 0
)

func main() {
	wg := sync.WaitGroup{}
	go HandleCallback(&wg)
	for i := 0; i < MaxExecutions; i++ {
		Semaphore <- struct{}{}
		wg.Add(1)
		go DoAction(i)
	}
	wg.Wait()
}

func HandleCallback(wg *sync.WaitGroup) {
	for {
		<-Callback
		for i := Current; i < MaxExecutions; i++ {

			if _, ok := CompletedExecs.Load(i); !ok {
				break
			}

			cb, _ := CompletedExecs.Swap(i, nil)
			cb.(func())()
			Current += 1
			<-Semaphore
			wg.Done()
		}
		if Current == MaxExecutions {
			fmt.Println("finished")
			break
		}
	}
}

func DoAction(i int) {
	rand.Intn(5)
	CompletedExecs.Store(i, func() {
		fmt.Println("Hello from", i)
	})
	Callback <- struct{}{}
}
