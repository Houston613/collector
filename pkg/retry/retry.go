package retry

import (
	"context"
	"time"
)

var Intervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func Do(ctx context.Context, fn func() error, isRetriable func(error) bool) error {
	var err error
	for i := 0; i <= len(Intervals); i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < len(Intervals) && isRetriable(err) {
			timer := time.NewTimer(Intervals[i])
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
