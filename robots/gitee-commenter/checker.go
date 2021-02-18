package main

import "sync"

type checker interface {
	check()
}

type checkerPool struct {
	ch chan checker
	wg sync.WaitGroup
}

func (cp *checkerPool) run(c checker) {
	cp.ch <- c
}

func (cp *checkerPool) shutdown() {
	close(cp.ch)
	cp.wg.Wait()
}

func newCheckerPool(maxGoroutine int) *checkerPool {
	p := &checkerPool{ch: make(chan checker)}
	p.wg.Add(maxGoroutine)
	for i := 0; i < maxGoroutine; i++ {
		go func() {
			defer p.wg.Done()
			for w := range p.ch {
				w.check()
			}
		}()
	}
	return p
}
