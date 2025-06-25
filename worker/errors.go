package worker

import "errors"

var (
	ErrPanic   = errors.New("task panicked")
	ErrTimeout = errors.New("task timed out")
)
