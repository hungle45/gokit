package worker

import (
	"context"
	"errors"
	"github.com/hungle45/gokit/queue"
	"math"
)

type Pool interface {
	Submit(runnable Runnable) Future[Void]
	SubmitResult(callable Callable[any]) Future[any]
}

type pool struct {
	ctx           context.Context
	maxWorker     int32
	blockingQueue queue.BlockingQueue[FutureTask[any]]
}

func (p pool) Submit(runnable Runnable) Future[Void] {
	//TODO implement me
	panic("implement me")
}

func (p pool) SubmitResult(callable Callable[any]) Future[any] {
	//TODO implement me
	panic("implement me")
}

func NewPool(maxWorker int32, options ...Option) Pool {
	if maxWorker == 0 {
		maxWorker = math.MaxInt32
	}

	if maxWorker < 0 {
		panic(errors.New("maxWorker must be greater than or equal to 0"))
	}

	config := &Config{
		ctx:       context.Background(),
		queueSize: math.MaxInt32,
	}

	for _, option := range options {
		option(config)
	}

	pool := &pool{
		ctx:       config.ctx,
		maxWorker: maxWorker,
	}

	return pool
}
