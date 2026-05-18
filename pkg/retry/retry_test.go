package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	t.Run("success first try", func(t *testing.T) {
		count := 0
		err := Do(context.Background(), func() error {
			count++
			return nil
		}, func(err error) bool { return true })
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("success after retries", func(t *testing.T) {
		intervals := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}

		count := 0
		err := DoWithIntervals(context.Background(), intervals, func() error {
			count++
			if count < 3 {
				return errors.New("retriable")
			}
			return nil
		}, func(err error) bool { return true })

		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("fail after all retries", func(t *testing.T) {
		intervals := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}

		count := 0
		expectedErr := errors.New("final error")
		err := DoWithIntervals(context.Background(), intervals, func() error {
			count++
			return expectedErr
		}, func(err error) bool { return true })

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 3, count) // 1 initial + 2 retries
	})

	t.Run("not retriable error", func(t *testing.T) {
		count := 0
		err := Do(context.Background(), func() error {
			count++
			return errors.New("not retriable")
		}, func(err error) bool {
			return false
		})

		assert.Error(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("context cancelled", func(t *testing.T) {
		intervals := []time.Duration{1 * time.Second}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		count := 0
		err := DoWithIntervals(ctx, intervals, func() error {
			count++
			return errors.New("retriable")
		}, func(err error) bool { return true })

		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, count)
	})
}
