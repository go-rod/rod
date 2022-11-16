package utils

import (
	"context"
	"fmt"
	mr "math/rand"
	"sync"
	"time"
)

// Sleep the goroutine for specified seconds, such as 2.3 seconds
func Sleep(seconds float64) {
	d := time.Duration(seconds * float64(time.Second))
	time.Sleep(d)
}

// Sleeper sleeps the current goroutine for sometime, returns the reason to wake, if ctx is done release resource
type Sleeper func(context.Context) error

// ErrMaxSleepCount type
type ErrMaxSleepCount struct {
	// Max count
	Max int
}

// Error interface
func (e *ErrMaxSleepCount) Error() string {
	return fmt.Sprintf("max sleep count %d exceeded", e.Max)
}

// Is interface
func (e *ErrMaxSleepCount) Is(err error) bool { _, ok := err.(*ErrMaxSleepCount); return ok }

// CountSleeper wakes immediately. When counts to the max returns *ErrMaxSleepCount
func CountSleeper(max int) Sleeper {
	l := sync.Mutex{}
	count := 0

	return func(ctx context.Context) error {
		l.Lock()
		defer l.Unlock()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if count == max {
			return &ErrMaxSleepCount{max}
		}
		count++
		return nil
	}
}

// DefaultBackoff algorithm: A(n) = A(n-1) * random[1.9, 2.1)
func DefaultBackoff(interval time.Duration) time.Duration {
	scale := 2 + (mr.Float64()-0.5)*0.2
	return time.Duration(float64(interval) * scale)
}

// BackoffSleeper returns a sleeper that sleeps in a backoff manner every time get called.
// The sleep interval of the sleeper will grow from initInterval to maxInterval by the specified algorithm, then use maxInterval as the interval.
// If maxInterval is not greater than 0, the sleeper will wake immediately.
// If algorithm is nil, DefaultBackoff will be used.
func BackoffSleeper(initInterval, maxInterval time.Duration, algorithm func(time.Duration) time.Duration) Sleeper {
	l := sync.Mutex{}

	if algorithm == nil {
		algorithm = DefaultBackoff
	}

	return func(ctx context.Context) error {
		l.Lock()
		defer l.Unlock()

		// wake immediately
		if maxInterval <= 0 {
			return nil
		}

		var interval time.Duration
		if initInterval < maxInterval {
			interval = algorithm(initInterval)
		} else {
			interval = maxInterval
		}

		t := time.NewTimer(interval)
		defer t.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			initInterval = interval
		}

		return nil
	}
}

// EachSleepers returns a sleeper wakes up when each sleeper is awake.
// If a sleeper returns error, it will wake up immediately.
func EachSleepers(list ...Sleeper) Sleeper {
	return func(ctx context.Context) (err error) {
		for _, s := range list {
			err = s(ctx)
			if err != nil {
				break
			}
		}

		return
	}
}

// RaceSleepers returns a sleeper wakes up when one of the sleepers wakes.
func RaceSleepers(list ...Sleeper) Sleeper {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		done := make(chan error, len(list))

		sleep := func(s Sleeper) {
			done <- s(ctx)
			cancel()
		}

		for _, s := range list {
			go sleep(s)
		}

		return <-done
	}
}

// Retry fn and sleeper until fn returns true or s returns error
func Retry(ctx context.Context, s Sleeper, fn func() (stop bool, err error)) error {
	for {
		stop, err := fn()
		if stop {
			return err
		}
		err = s(ctx)
		if err != nil {
			return err
		}
	}
}
