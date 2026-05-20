package retry

import (
	"context"
	"time"
)

var defaultIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// обертка с дефолтными интервалами для повторов
func Do(ctx context.Context, fn func() error, isRetriable func(error) bool) error {
	return retry(ctx, defaultIntervals, fn, isRetriable)
}

// обертка с кастомыми интервалами для повторов
func DoWithIntervals(ctx context.Context, intervals []time.Duration, fn func() error, isRetriable func(error) bool) error {
	return retry(ctx, intervals, fn, isRetriable)
}

// логика повторов с учетом контекста и проверкой на возможность повторения ошибки
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
