package ops

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// spread does the same work for many files or devices on as many
// goroutines as the machine has processors, while the caller takes the
// results one after the other in their order: what comes out does not
// depend on which piece of work ended first (spec 16).
type spread struct {
	done    []chan struct{}
	next    atomic.Int64
	stopped atomic.Bool
	wg      sync.WaitGroup
}

// newSpread starts work(worker, i) for i = 0 to n-1. worker numbers the
// goroutine that does the work, from 0 to workers(n)-1: what must not be
// shared between goroutines is kept once per worker.
func newSpread(n int, work func(worker, i int)) *spread {
	s := &spread{done: make([]chan struct{}, n)}
	for i := range s.done {
		s.done[i] = make(chan struct{})
	}
	for w := 0; w < workers(n); w++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				i := int(s.next.Add(1)) - 1
				if i >= n {
					return
				}
				if !s.stopped.Load() {
					work(w, i)
				}
				close(s.done[i])
			}
		}()
	}
	return s
}

// workers is how many goroutines do n pieces of work.
func workers(n int) int {
	w := runtime.GOMAXPROCS(0)
	if w > n {
		w = n
	}
	if w < 1 {
		w = 1
	}
	return w
}

// wait waits until piece i is done.
func (s *spread) wait(i int) { <-s.done[i] }

// stop ends the work: the pieces not begun are left out, and the ones
// under way are waited for. To be called before the caller returns.
func (s *spread) stop() {
	s.stopped.Store(true)
	s.wg.Wait()
}
