package queue

import "errors"

var (
	// ErrQueueIsEmpty is an error returned whenever the queue is empty and there
	ErrQueueIsEmpty = errors.New("queue is empty")

	// ErrQueueIsFull is an error returned whenever the queue is full and there
	// is an attempt to add an element to it.
	ErrQueueIsFull = errors.New("queue is full")
)
