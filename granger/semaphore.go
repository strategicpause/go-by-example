package main

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
