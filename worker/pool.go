package worker

import (
	"context"
	"errors"
	"github.com/hungle45/gokit/queue"
	"math"
	"sync"
	"sync/atomic"
)

const DefaultQueueSize = 10

var (
	_ Pool = &pool{}
)

type Pool interface {
	// Stopped returns true if the pool has been stopped.
	Stopped() bool
	// RunningWorkers returns the number of workers currently running in the pool.
	RunningWorkers() int32
	// WaitingTasks returns the number of tasks currently waiting in the queue.
	WaitingTasks() int32

	// Submit submits a runnable task to the pool. If the pool is full, it blocks until a worker is available.
	Submit(runnable Function) Future[Void]
	// TrySubmit attempts to submit a runnable task to the pool without blocking.
	TrySubmit(runnable Function) (Future[Void], bool)

	// SubmitResult submits a callable task to the pool. If the pool is full, it blocks until a worker is available.
	SubmitResult(callable Supplier[any]) Future[any]
	// TrySubmitResult attempts to submit a callable task to the pool without blocking.
	TrySubmitResult(callable Supplier[any]) (Future[any], bool)

	// Stop stops the pool and waits for all workers to finish. Returns a Future that completes when the pool is stopped.
	Stop() Future[Void]
	// StopAndWait stops the pool and waits for all workers to finish. It blocks until the pool is stopped.
	StopAndWait()
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
		queueSize: DefaultQueueSize,
	}

	for _, option := range options {
		option(config)
	}

	pool := &pool{
		ctx:           config.ctx,
		maxWorker:     maxWorker,
		blockingQueue: queue.NewBlockingQueue([]Runner{}, queue.WithCapacity(int(config.queueSize))),
		submitWaiters: make(chan struct{}, 1),
	}

	pool.ctx, pool.cancel = context.WithCancelCause(pool.ctx)

	return pool
}

type pool struct {
	ctx           context.Context
	cancel        context.CancelCauseFunc
	mutex         sync.Mutex
	maxWorker     int32
	blockingQueue queue.BlockingQueue[Runner]

	workerCount     atomic.Int32
	workerWaitGroup sync.WaitGroup
	submitWaiters   chan struct{}
}

func (p *pool) Stopped() bool {
	return p.ctx.Err() != nil
}

func (p *pool) RunningWorkers() int32 {
	return p.workerCount.Load()
}

func (p *pool) WaitingTasks() int32 {
	return int32(p.blockingQueue.Size())
}

func (p *pool) Submit(task Function) Future[Void] {
	if p.Stopped() {
		return NewFutureTask(p.ctx, func() (Void, error) { return Empty, ErrPoolStopped })
	}

	futureTask := NewFutureTask(p.ctx, func() (Void, error) {
		return Empty, task()
	})

	err := p.submit(futureTask)
	if err != nil {
		futureTask.CancelWithErr(err)
	}

	return futureTask
}

func (p *pool) TrySubmit(task Function) (Future[Void], bool) {
	if p.Stopped() {
		return NewFutureTask(p.ctx, func() (Void, error) { return Empty, ErrPoolStopped }), false
	}

	futureTask := NewFutureTask(p.ctx, func() (Void, error) {
		return Empty, task()
	})

	err := p.trySubmit(futureTask)
	if err != nil {
		futureTask.CancelWithErr(err)
		return futureTask, false
	}

	return futureTask, true
}

func (p *pool) SubmitResult(task Supplier[any]) Future[any] {
	if p.Stopped() {
		return NewFutureTask(p.ctx, func() (any, error) { return Empty, ErrPoolStopped })
	}

	futureTask := NewFutureTask(p.ctx, task)

	err := p.submit(futureTask)
	if err != nil {
		futureTask.CancelWithErr(err)
	}

	return futureTask
}

func (p *pool) TrySubmitResult(callable Supplier[any]) (Future[any], bool) {
	if p.Stopped() {
		return NewFutureTask(p.ctx, func() (any, error) { return Empty, ErrPoolStopped }), false
	}

	futureTask := NewFutureTask(p.ctx, callable)

	err := p.trySubmit(futureTask)
	if err != nil {
		futureTask.CancelWithErr(err)
		return futureTask, false
	}

	return futureTask, true
}

func (p *pool) Stop() Future[Void] {
	task := NewFutureTask(context.Background(), func() (Void, error) {
		if !p.Stopped() {
			p.workerWaitGroup.Wait()
			p.cancel(ErrPoolStopped)
		}

		return Empty, nil
	})

	go task.Run()

	return task
}

func (p *pool) StopAndWait() {
	_, _ = p.Stop().Get()
}

func (p *pool) submit(task Runner) error {
	for {
		if err := p.trySubmit(task); !errors.Is(err, ErrPoolFull) {
			return err
		}

		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-p.submitWaiters:
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			default:
			}
		}
	}
}

func (p *pool) trySubmit(task Runner) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.workerCount.Load() < p.maxWorker {
		p.workerCount.Add(1)
		p.workerWaitGroup.Add(1)
		go p.worker(task)
		return nil
	}

	err := p.blockingQueue.Offer(task)
	if err != nil {
		if errors.Is(err, queue.ErrQueueIsFull) {
			return ErrPoolFull
		}
		return err
	}

	p.notifySubmitWaiters()
	return nil
}

func (p *pool) worker(task Runner) {
	var readTaskErr error
	for {
		if task != nil {
			task.Run()
		}

		task, readTaskErr = p.readTask()
		if readTaskErr != nil {
			return
		}
	}
}

func (p *pool) readTask() (Runner, error) {
	select {
	case <-p.ctx.Done():
		p.workerCount.Add(-1)
		p.workerWaitGroup.Done()
		return nil, p.ctx.Err()
	default:
	}

	if p.workerCount.Load() > p.maxWorker {
		p.workerCount.Add(-1)
		p.workerWaitGroup.Done()
		return nil, ErrMaxWorkerReached
	}

	task, err := p.blockingQueue.Get()
	if err != nil {
		p.workerCount.Add(-1)
		p.workerWaitGroup.Done()

		if errors.Is(err, queue.ErrQueueIsEmpty) {
			p.notifySubmitWaiters()
		}

		return nil, err
	}

	p.notifySubmitWaiters()
	return task, nil
}

func (p *pool) notifySubmitWaiters() {
	select {
	case p.submitWaiters <- struct{}{}:
	default:
	}
}
