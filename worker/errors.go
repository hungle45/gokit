package worker

import "errors"

var (
	ErrPanic            = errors.New("task panicked")
	ErrTimeout          = errors.New("task timed out")
	ErrPoolStopped      = errors.New("worker pool has been stopped")
	ErrPoolFull         = errors.New("worker pool is full")
	ErrTaskCancelled    = errors.New("task was cancelled")
	ErrMaxWorkerReached = errors.New("maximum worker limit reached")
)
