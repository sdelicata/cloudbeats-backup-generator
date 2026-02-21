// Package worker provides a concurrent worker pool for parallel task execution.
package worker

import (
	"context"
	"sync"
	"sync/atomic"
)

// ProgressFunc is called after each item is processed with (done, total).
type ProgressFunc func(done, total int)

// Process runs fn on each item using n concurrent goroutines.
// Results are returned in the same order as items. Errors are collected per-item.
func Process[T any, R any](ctx context.Context, items []T, n int, fn func(context.Context, T) (R, error), progress ProgressFunc) ([]R, []error) {
	total := len(items)
	if total == 0 {
		return nil, nil
	}

	results := make([]R, total)
	errors := make([]error, total)

	var done atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, n)

	for i, item := range items {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, it T) {
			defer wg.Done()
			defer func() { <-sem }()

			r, err := fn(ctx, it)
			results[idx] = r
			errors[idx] = err

			current := int(done.Add(1))
			if progress != nil {
				progress(current, total)
			}
		}(i, item)
	}

	wg.Wait()

	return results, errors
}
