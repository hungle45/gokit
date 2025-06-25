package worker

type Void struct{}

type Runnable func() error

type Callable[R any] func() (R, error)

type Callback[T any] func(T, error)
