package ggmat

import "sync"

// parFor runs fn(i) for i in [0,n) with up to workers goroutines.
func parFor(n, workers int, fn func(i int)) {
	if n == 0 {
		return
	}
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for i := s; i < e; i++ {
				fn(i)
			}
		}(start, end)
	}
	wg.Wait()
}
