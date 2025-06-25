package worker

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

type Future[T any] interface {
	// Done returns a channel that is closed when the future is complete or has failed.
	Done() <-chan struct{}

	// Get waits for the future to complete and returns the result or an error if it failed.
	Get() (T, error)

	// GetWithTimeOut waits for the future to complete and returns the result or an error if it times out.
	GetWithTimeOut(timeout time.Duration) (T, error)
}

type FutureTask[R any] struct {
	ctx      context.Context
	callback func(value R, err error)
	callable Callable[R]
}

func NewFutureTask[R any](ctx context.Context, callable Callable[R]) *FutureTask[R] {
	childCtx, cancel := context.WithCancelCause(ctx)
	return &FutureTask[R]{
		ctx:      childCtx,
		callable: callable,
		callback: func(result R, err error) {
			cancel(&futureResolution[R]{
				value: result,
				err:   err,
			})
		},
	}
}

func (f *FutureTask[R]) Done() <-chan struct{} {
	return f.ctx.Done()
}

func (f *FutureTask[R]) GetWithTimeOut(timeout time.Duration) (R, error) {
	select {
	case <-f.ctx.Done():
		return f.Get()
	case <-time.After(timeout):
		var zero R
		return zero, ErrTimeout
	}
}

func (f *FutureTask[R]) Get() (R, error) {
	<-f.ctx.Done()

	cause := context.Cause(f.ctx)
	if cause != nil {
		var resolution *futureResolution[R]
		if errors.As(cause, &resolution) {
			return resolution.value, resolution.err
		}
	}

	var zero R
	return zero, cause
}

func (f *FutureTask[R]) Run() {
	result, err := f.invoke()
	f.callback(result, err)
	return
}

func (f *FutureTask[R]) invoke() (result R, err error) {
	defer func() {
		if p := recover(); p != nil {
			if e, ok := p.(error); ok {
				err = fmt.Errorf("%w: %w, %s", ErrPanic, e, debug.Stack())
			} else {
				err = fmt.Errorf("%w: %v, %s", ErrPanic, p, debug.Stack())
			}
			return
		}
	}()

	return f.callable()
}

type futureResolution[V any] struct {
	value V
	err   error
}

func (v *futureResolution[V]) Error() string {
	if v.err != nil {
		return v.err.Error()
	}
	return fmt.Sprintf("future resolved: %v", v.value)
}
