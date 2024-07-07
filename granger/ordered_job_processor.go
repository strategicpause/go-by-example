package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

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
	// callbackCh indicates that an action has completed and that we should
	// start processing callbacks.
	callbackCh chan struct{}
	// stopCh signals that we are done processing incoming jobs
	stopCh chan struct{}
	// semaphore allows only n goroutines to run at once
	semaphore *Semaphore
	// wg ensures all goroutines are done before stopping.
	wg sync.WaitGroup
}

func NewOrderedJobProcessor(parallelization int) *OrderedJobProcessor {
	ojp := &OrderedJobProcessor{
		currentExec:    0,
		maxExec:        atomic.Int64{},
		completedExecs: &sync.Map{},
		callbackCh:     make(chan struct{}),
		stopCh:         make(chan struct{}),
		semaphore:      NewSemaphore(parallelization),
		wg:             sync.WaitGroup{},
	}
	go ojp.start()

	return ojp
}

func (o *OrderedJobProcessor) start() {
	for {
		select {
		case <-o.callbackCh:
			for i := o.currentExec; i < o.maxExec.Load(); i++ {
				// We're still waiting on the next job to finish.
				if _, ok := o.completedExecs.Load(i); !ok {
					break
				}
				cb, _ := o.completedExecs.Swap(i, nil)
				err := cb.(func() error)()
				if err != nil {
					panic(err)
				}
				o.currentExec += 1
				o.wg.Done()
				o.semaphore.Release()
			}
		case <-o.stopCh:
			fmt.Println("finished")
			break
		}
	}
}

func (o *OrderedJobProcessor) Stop() {
	o.wg.Wait()
	o.stopCh <- struct{}{}
}

func (o *OrderedJobProcessor) SubmitJob(f func() error, cb func() error) {
	o.semaphore.Acquire()

	action := &action{
		i:              o.maxExec.Load(),
		fn:             f,
		cb:             cb,
		completedExecs: o.completedExecs,
		callback:       o.callbackCh,
	}

	o.maxExec.Add(1)
	o.wg.Add(1)
	go action.Start()
}

type action struct {
	i              int64
	fn             func() error
	cb             func() error
	completedExecs *sync.Map
	callback       chan struct{}
}

func (a *action) Start() {
	err := a.fn()
	if err != nil {
		panic(err)
	}
	a.completedExecs.Store(a.i, a.cb)
	a.callback <- struct{}{}
}
