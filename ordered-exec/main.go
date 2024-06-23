package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Parallelization answers how many goroutines should we run at once.
	Parallelization = 4
	// MaxExecutions answers how many total jobs to run.
	MaxExecutions = 100
)

func main() {
	ojp := NewOrderedJobProcessor(Parallelization)
	ojp.Start()
	for i := 0; i < MaxExecutions; i++ {
		a := func(i int64) {
			fmt.Println("Starting", i)
			sleepTime := time.Duration(rand.Int63n(3)) * time.Second
			time.Sleep(sleepTime)
		}
		cb := func(i int64) {
			fmt.Println("Hello from", i)
		}
		ojp.SubmitJob(a, cb)
	}
	ojp.Stop()
}

type Semaphore struct {
	semaphore chan struct{}
}

func NewSemaphore(n int) *Semaphore {
	return &Semaphore{semaphore: make(chan struct{}, n)}
}

func (s *Semaphore) Acquire() {
	s.semaphore <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.semaphore
}

// OrderedJobProcessor will process jobs in the order that they are submitted.
type OrderedJobProcessor struct {
	// currentExec tells us which execution we need to process next.
	currentExec int64
	// maxExec is a monotonically increasing value which helps us order jobs. if a < b, then
	// job(a)'s callback will be started before job(b).
	maxExec atomic.Int64
	// completedExecs stores callback functions for actions which have finished
	// executing.
	completedExecs *sync.Map
	// callback indicates that an action has completed and that we should
	// start processing callbacks.
	callback chan struct{}
	// stop signals that we are done processing incoming jobs
	stop chan struct{}
	// semaphore allows only n goroutines to run at once
	semaphore *Semaphore
	// wg ensures all goroutines are done before stopping.
	wg sync.WaitGroup
}

func NewOrderedJobProcessor(parallelization int) *OrderedJobProcessor {
	return &OrderedJobProcessor{
		currentExec:    0,
		maxExec:        atomic.Int64{},
		completedExecs: &sync.Map{},
		callback:       make(chan struct{}),
		stop:           make(chan struct{}),
		semaphore:      NewSemaphore(parallelization),
		wg:             sync.WaitGroup{},
	}
}
func (o *OrderedJobProcessor) Start() {
	go o.start()
}

func (o *OrderedJobProcessor) start() {
	for {
		select {
		case <-o.callback:
			for i := o.currentExec; i < o.maxExec.Load(); i++ {
				if _, ok := o.completedExecs.Load(i); !ok {
					break
				}
				o.wg.Done()
				cb, _ := o.completedExecs.Swap(i, nil)
				cb.(func(int64))(i)
				o.currentExec += 1
				o.semaphore.Release()
			}
		case <-o.stop:
			fmt.Println("finished")
			break
		}
	}
}

func (o *OrderedJobProcessor) Stop() {
	o.wg.Wait()
	o.stop <- struct{}{}
}

func (o *OrderedJobProcessor) SubmitJob(f func(int64), cb func(int64)) {
	o.semaphore.Acquire()

	action := &action{
		i:              o.maxExec.Load(),
		fn:             f,
		cb:             cb,
		completedExecs: o.completedExecs,
		callback:       o.callback,
	}

	o.maxExec.Add(1)
	o.wg.Add(1)
	go action.Start()
}

type action struct {
	i              int64
	fn             func(int64)
	cb             func(int64)
	completedExecs *sync.Map
	callback       chan struct{}
}

func (a *action) Start() {
	a.fn(a.i)
	a.completedExecs.Store(a.i, a.cb)
	a.callback <- struct{}{}
}
