package retry

import (
	"context"
	"time"
)

var defaultIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do executes the given function fn, retrying on retriable errors using default intervals.
func Do(ctx context.Context, fn func() error, isRetriable func(error) bool) error {
	return retry(ctx, defaultIntervals, fn, isRetriable)
}

// DoWithIntervals executes the given function fn, retrying on retriable errors using custom intervals.
func DoWithIntervals(ctx context.Context, intervals []time.Duration, fn func() error, isRetriable func(error) bool) error {
	return retry(ctx, intervals, fn, isRetriable)
}

// retry handles the retry loop with context cancellation and error checking.
func retry(ctx context.Context, intervals []time.Duration, fn func() error, isRetriable func(error) bool) error {
	var err error
	for i := 0; i <= len(intervals); i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < len(intervals) && isRetriable(err) {
			timer := time.NewTimer(intervals[i])
			select {
			case <-timer.C:
				continue
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
		break
	}
	return err
}
