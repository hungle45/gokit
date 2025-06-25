package queue

import (
	"encoding/json"
	"sync"
)

const DefaultBlockingQueueCapacity = 10

// BlockingQueue represents a thread-safe, blocking queue.
type BlockingQueue[T any] interface {
	// OfferWait adds an element to the queue, blocking if the queue is full.
	OfferWait(elem T)
	// Offer adds an element to the queue without blocking. Returns an error if the queue is full.
	Offer(elem T) error
	// GetWait retrieves and removes the head of the queue, blocking if the queue is empty.
	GetWait() T
	// Get retrieves and removes the head of the queue without blocking. Returns an error if the queue is empty.
	Get() (T, error)
	// Clear removes all elements from the queue and returns them.
	Clear() []T
	// Iterator returns a channel that yields all elements currently in the queue.
	Iterator() <-chan T
	// Peek retrieves, but does not remove, the head of the queue. Returns an error if the queue is empty.
	Peek() (T, error)
	// PeekWait retrieves, but does not remove, the head of the queue, blocking if the queue is empty.
	PeekWait() T
	// Size returns the number of elements in the queue.
	Size() int
	// IsEmpty returns true if the queue contains no elements.
	IsEmpty() bool
	// MarshalJSON implements the json.Marshaler interface.
	MarshalJSON() ([]byte, error)
}

// NewBlockingQueue creates and returns a new BlockingQueue.
func NewBlockingQueue[T any](initItems []T, opts ...Option) BlockingQueue[T] {
	configs := Config{
		Capacity: DefaultBlockingQueueCapacity,
	}

	for _, opt := range opts {
		opt(&configs)
	}

	queue := &blockingQueue[T]{
		items:    make([]T, 0, configs.Capacity),
		capacity: configs.Capacity,
	}

	for _, item := range initItems {
		if len(queue.items) < queue.capacity {
			queue.items = append(queue.items, item)
		} else {
			break
		}
	}

	queue.notEmpty = sync.NewCond(&queue.lock)
	queue.notFull = sync.NewCond(&queue.lock)

	return queue
}

type blockingQueue[T any] struct {
	items    []T
	capacity int

	lock     sync.RWMutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
}

func (bq *blockingQueue[T]) OfferWait(item T) {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	for bq.isFull() {
		bq.notFull.Wait()
	}

	bq.items = append(bq.items, item)
	bq.notEmpty.Signal()
}

func (bq *blockingQueue[T]) Offer(item T) error {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	if bq.isFull() {
		return ErrQueueIsFull
	}

	bq.items = append(bq.items, item)
	bq.notEmpty.Signal()

	return nil
}

func (bq *blockingQueue[T]) GetWait() (v T) {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	for bq.isEmpty() {
		bq.notEmpty.Wait()
	}

	val, _ := bq.get()
	return val
}

func (bq *blockingQueue[T]) Get() (v T, err error) {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	return bq.get()
}

func (bq *blockingQueue[T]) Clear() []T {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	if bq.isEmpty() {
		return nil
	}

	removed := make([]T, len(bq.items))
	copy(removed, bq.items)
	bq.items = bq.items[:0]

	bq.notFull.Broadcast()
	bq.notEmpty.Broadcast()

	return removed
}

func (bq *blockingQueue[T]) Iterator() <-chan T {
	bq.lock.RLock()
	snapshot := make([]T, len(bq.items))
	copy(snapshot, bq.items)
	bq.lock.RUnlock()

	iteratorCh := make(chan T, len(snapshot))

	go func() {
		defer close(iteratorCh)
		for _, elem := range snapshot {
			iteratorCh <- elem
		}
	}()

	return iteratorCh
}

func (bq *blockingQueue[T]) Peek() (v T, err error) {
	bq.lock.RLock()
	defer bq.lock.RUnlock()

	if bq.isEmpty() {
		return v, ErrQueueIsEmpty
	}

	return bq.items[0], nil
}

func (bq *blockingQueue[T]) PeekWait() T {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	for bq.isEmpty() {
		bq.notEmpty.Wait()
	}

	return bq.items[0]
}

func (bq *blockingQueue[T]) Size() int {
	bq.lock.RLock()
	defer bq.lock.RUnlock()

	return len(bq.items)
}

func (bq *blockingQueue[T]) IsEmpty() bool {
	bq.lock.RLock()
	defer bq.lock.RUnlock()

	return bq.isEmpty()
}

func (bq *blockingQueue[T]) MarshalJSON() ([]byte, error) {
	bq.lock.RLock()
	defer bq.lock.RUnlock()

	if bq.isEmpty() {
		return []byte("[]"), nil
	}

	return json.Marshal(bq.items)
}

func (bq *blockingQueue[T]) isEmpty() bool {
	return len(bq.items) == 0
}

func (bq *blockingQueue[T]) isFull() bool {
	return len(bq.items) >= bq.capacity
}

func (bq *blockingQueue[T]) get() (v T, err error) {
	if bq.isEmpty() {
		return v, ErrQueueIsEmpty
	}

	elem := bq.items[0]
	bq.items = bq.items[1:]

	bq.notFull.Signal()

	return elem, nil
}
