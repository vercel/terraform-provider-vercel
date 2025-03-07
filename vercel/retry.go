package vercel

import (
	"math"
	"time"
)

type Retry struct {
	Base     time.Duration
	Max      time.Duration
	Attempts int
}

type RetryFunc func(attempt int) (shouldRetry bool, err error)

func (r *Retry) Do(fn RetryFunc) error {
	for attempt := 1; attempt <= r.Attempts; attempt++ {
		shouldRetry, err := fn(attempt)

		if err == nil {
			return nil
		} else if !shouldRetry || attempt == r.Attempts {
			return err
		}

		sleepDuration := r.getSleepDuration(attempt)
		time.Sleep(sleepDuration)
	}

	return nil
}

func (r *Retry) getSleepDuration(attempt int) time.Duration {
	// exponential backoff
	backoff := float64(r.Base) * math.Pow(2, float64(attempt-1))
	sleep := time.Duration(backoff)

	return sleep
}
