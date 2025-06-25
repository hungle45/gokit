package worker

type Void struct{}

var Empty = Void{}

type Runner interface {
	Run()
}

type Function func() error

type Supplier[R any] func() (R, error)

type Callback[T any] func(T, error)
