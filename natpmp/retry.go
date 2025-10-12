package natpmp

import (
	"fmt"
	"time"
)

// retry will retry the
type retry struct {
	initial        time.Duration
	timeout        time.Duration
	maxRetries     int
	retryImmediate func(error) bool
	retryDelay     func(error) bool
}

func (r *retry) run(fn func(deadline time.Time) error) error {
	var finalDeadline time.Time
	if r.timeout != 0 {
		finalDeadline = time.Now().Add(r.timeout)
	}
	nextDeadline := time.Now().Add(initialPause)

	var tries uint
	for tries = 0; (tries < maxRetries && finalDeadline.IsZero()) || time.Now().Before(finalDeadline); {
		err := fn(minTime(nextDeadline, finalDeadline))
		if r.retryImmediate != nil && r.retryImmediate(err) {
			continue
		}
		if r.retryDelay != nil && r.retryDelay(err) {
			tries++
			nextDeadline = time.Now().Add(initialPause * 1 << tries)
			continue
		}
		return err
	}
	return fmt.Errorf("Timed out trying to contact gateway")
}

func minTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	if a.Before(b) {
		return a
	}
	return b
}
