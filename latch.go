package main

import (
	"sync"
)

type Latch struct {
	cond *sync.Cond
	open bool
}

func NewLatch() *Latch {
	return &Latch{cond: sync.NewCond(&sync.Mutex{})}
}

func (l *Latch) Wait() {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	for !l.open {
		l.cond.Wait()
	}
	l.open = false
}

func (l *Latch) Open() {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	l.open = true
	l.cond.Signal()
}
