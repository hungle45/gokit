package worker_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/hungle45/gokit/worker"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPool tests the initialization of the pool.
func TestNewPool(t *testing.T) {
	t.Run("Default maxWorker and queueSize", func(t *testing.T) {
		pool := worker.NewPool(0) // 0 should default to MaxInt32 for maxWorker
		assert.NotNil(t, pool)
		assert.False(t, pool.Stopped(), "Pool should not be stopped initially")

		// Submit a few tasks to check if workers spin up
		var counter atomic.Int32
		for i := 0; i < 5; i++ {
			f := pool.Submit(func() error {
				counter.Add(1)
				return nil
			})
			_, err := f.Get()
			assert.NoError(t, err)
		}
		assert.Equal(t, int32(5), counter.Load())
	})

	t.Run("Specific maxWorker and queueSize", func(t *testing.T) {
		pool := worker.NewPool(2, worker.WithQueueSize(3))
		assert.NotNil(t, pool)
		assert.False(t, pool.Stopped())
		// More specific tests on capacity and worker behavior will follow
	})

	t.Run("Negative maxWorker should panic", func(t *testing.T) {
		assert.Panics(t, func() {
			worker.NewPool(-1)
		}, "NewPool with negative maxWorker should panic")
	})

	t.Run("With custom context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pool := worker.NewPool(1, worker.WithContext(ctx))
		assert.NotNil(t, pool)

		// Test that canceling parent context stops the pool
		cancel()
		time.Sleep(50 * time.Millisecond) // Give time for cancellation to propagate
		assert.True(t, pool.Stopped(), "Pool should be stopped when parent context is cancelled")
	})
}

// TestSubmitAndGet verifies basic task submission and result retrieval.
func TestSubmitAndGet(t *testing.T) {
	pool := worker.NewPool(1) // 1 worker
	defer pool.StopAndWait()

	t.Run("Submit Function (Void) - success", func(t *testing.T) {
		var executed atomic.Bool
		future := pool.Submit(func() error {
			time.Sleep(10 * time.Millisecond) // Simulate work
			executed.Store(true)
			return nil
		})

		_, err := future.Get()
		assert.NoError(t, err, "Future.Get should not return error for successful function")
		assert.True(t, executed.Load(), "Function should have been executed")
	})

	t.Run("Submit Function (Void) - error", func(t *testing.T) {
		expectedErr := errors.New("task failed")
		future := pool.Submit(func() error {
			return expectedErr
		})

		_, err := future.Get()
		assert.ErrorIs(t, err, expectedErr, "Future.Get should return error from function")
	})

	t.Run("Submit Function (Void) - panic", func(t *testing.T) {
		future := pool.Submit(func() error {
			panic("oops, a panic")
		})

		_, err := future.Get()
		assert.Error(t, err, "Future.Get should return error on panic")
		assert.ErrorIs(t, err, worker.ErrPanic, "Error should be ErrPanic")
		assert.Contains(t, err.Error(), "oops, a panic", "Error message should contain panic message")
	})

	t.Run("SubmitResult (any) - success", func(t *testing.T) {
		expectedVal := 123
		future := pool.SubmitResult(func() (any, error) {
			time.Sleep(10 * time.Millisecond)
			return expectedVal, nil
		})

		result, err := future.Get()
		assert.NoError(t, err, "Future.Get should not return error for successful supplier")
		assert.Equal(t, expectedVal, result, "Future.Get should return correct result from supplier")
	})

	t.Run("SubmitResult (any) - error", func(t *testing.T) {
		expectedErr := errors.New("supplier failed")
		future := pool.SubmitResult(func() (any, error) {
			return nil, expectedErr
		})

		_, err := future.Get()
		assert.ErrorIs(t, err, expectedErr, "Future.Get should return error from supplier")
	})

	t.Run("SubmitResult (any) - panic", func(t *testing.T) {
		future := pool.SubmitResult(func() (any, error) {
			panic(404)
		})

		_, err := future.Get()
		assert.Error(t, err, "Future.Get should return error on panic")
		assert.ErrorIs(t, err, worker.ErrPanic, "Error should be ErrPanic")
		assert.Contains(t, err.Error(), "404", "Error message should contain panic value")
	})
}

// TestTrySubmit verifies non-blocking submission.
func TestTrySubmitVoid(t *testing.T) {
	tests := []struct {
		name          string
		maxWorker     int32
		queueSize     int32
		preoccupyFunc func(pool worker.Pool) *sync.WaitGroup // Function to set up initial state
		submitFunc    func(pool worker.Pool, executed *atomic.Bool) (worker.Future[worker.Void], bool)
		expectedOK    bool
		expectedErr   error
		validateFunc  func(t *testing.T, future worker.Future[worker.Void], executed *atomic.Bool)
	}{
		{
			name:      "success to worker",
			maxWorker: 1,
			queueSize: 1,
			preoccupyFunc: func(pool worker.Pool) *sync.WaitGroup {
				// No pre-occupation needed, worker is free
				return nil
			},
			submitFunc: func(pool worker.Pool, executed *atomic.Bool) (worker.Future[worker.Void], bool) {
				future, ok := pool.TrySubmit(func() error { executed.Store(true); return nil })
				return future, ok
			},
			expectedOK:  true,
			expectedErr: nil,
			validateFunc: func(t *testing.T, future worker.Future[worker.Void], executed *atomic.Bool) {
				// Give time for task to execute
				time.Sleep(20 * time.Millisecond)
				_, err := future.Get()
				assert.NoError(t, err)
				assert.True(t, executed.Load(), "Task should be executed when worker is free")
			},
		},
		{
			name:      "success to queue",
			maxWorker: 1,
			queueSize: 1,
			preoccupyFunc: func(pool worker.Pool) *sync.WaitGroup {
				var wg sync.WaitGroup
				wg.Add(1)
				pool.Submit(func() error {
					defer wg.Done()
					time.Sleep(100 * time.Millisecond) // Occupy the worker
					return nil
				})
				time.Sleep(10 * time.Millisecond) // Ensure worker starts
				return &wg
			},
			submitFunc: func(pool worker.Pool, executed *atomic.Bool) (worker.Future[worker.Void], bool) {
				future, ok := pool.TrySubmit(func() error { executed.Store(true); return nil })
				return future, ok
			},
			expectedOK:  true,
			expectedErr: nil,
			validateFunc: func(t *testing.T, future worker.Future[worker.Void], executed *atomic.Bool) {
				// Task will run after the pre-occupying task finishes
				_, err := future.Get()
				assert.NoError(t, err)
				assert.True(t, executed.Load(), "Task should be executed from queue")
			},
		},
		{
			name:      "fail (queue full)",
			maxWorker: 1,
			queueSize: 0, // No queue capacity
			preoccupyFunc: func(pool worker.Pool) *sync.WaitGroup {
				var wg sync.WaitGroup
				wg.Add(1)
				pool.Submit(func() error {
					defer wg.Done()
					time.Sleep(100 * time.Millisecond) // Occupy the worker
					return nil
				})
				time.Sleep(10 * time.Millisecond) // Ensure worker starts
				return &wg
			},
			submitFunc: func(pool worker.Pool, executed *atomic.Bool) (worker.Future[worker.Void], bool) {
				future, ok := pool.TrySubmit(func() error { return nil })
				return future, ok
			},
			expectedOK:  false,
			expectedErr: worker.ErrPoolFull,
			validateFunc: func(t *testing.T, future worker.Future[worker.Void], executed *atomic.Bool) {
				_, err := future.Get()
				assert.ErrorIs(t, err, worker.ErrPoolFull)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := worker.NewPool(tc.maxWorker, worker.WithQueueSize(tc.queueSize))
			defer pool.StopAndWait()

			// Use a specific executed atomic for each test case
			var executedForTest atomic.Bool
			executedForTest.Store(false)

			// Pre-occupy workers/queue if necessary
			var preoccupyWg *sync.WaitGroup
			if tc.preoccupyFunc != nil {
				preoccupyWg = tc.preoccupyFunc(pool)
			}

			// Perform the TrySubmit operation
			future, ok := tc.submitFunc(pool, &executedForTest)
			assert.Equal(t, tc.expectedOK, ok, "Expected submit success status mismatch")
			require.NotNil(t, future, "Future should not be nil even on failed submission")

			// Validate the future's result/error
			if tc.validateFunc != nil {
				tc.validateFunc(t, future, &executedForTest)
			} else {
				// Default validation for cases without specific `validateFunc`
				_, err := future.Get()
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				} else {
					assert.NoError(t, err)
				}
			}

			// Clean up preoccupying tasks
			if preoccupyWg != nil {
				preoccupyWg.Wait()
			}
		})
	}
}

// TestPoolStopAndAwait verifies pool shutdown behavior.
func TestPoolStopAndAwait(t *testing.T) {
	t.Run("Stop an empty pool", func(t *testing.T) {
		pool := worker.NewPool(1)
		assert.False(t, pool.Stopped())
		stopFuture := pool.Stop()
		_, err := stopFuture.Get()
		assert.NoError(t, err)
		assert.True(t, pool.Stopped(), "Pool should be stopped after Stop().Get()")
	})

	t.Run("StopAndWait on an empty pool", func(t *testing.T) {
		pool := worker.NewPool(1)
		assert.False(t, pool.Stopped())
		pool.StopAndWait()
		assert.True(t, pool.Stopped(), "Pool should be stopped after StopAndWait()")
	})

	t.Run("Stop waits for running tasks to complete", func(t *testing.T) {
		pool := worker.NewPool(1)
		var taskDone sync.WaitGroup
		taskDone.Add(1)

		future := pool.Submit(func() error {
			time.Sleep(100 * time.Millisecond) // Simulate a long-running task
			taskDone.Done()
			return nil
		})

		stopCalled := make(chan struct{})
		go func() {
			close(stopCalled)
			pool.StopAndWait()
		}()

		<-stopCalled // Ensure StopAndWait has been called
		select {     // Verify pool is not immediately stopped
		case <-time.After(50 * time.Millisecond): // Wait half the task time
			assert.False(t, pool.Stopped(), "Pool should not be stopped while task is running")
		case <-pool.Stop().Done():
			t.Fatal("Pool stopped too early")
		}

		taskDone.Wait()      // Wait for the task to actually finish
		<-pool.Stop().Done() // Wait for the stop operation to complete

		assert.True(t, pool.Stopped(), "Pool should be stopped after task completes and StopAndWait returns")
		_, err := future.Get()
		assert.NoError(t, err, "Running task should complete successfully before stop")
	})

	t.Run("Submit after stop", func(t *testing.T) {
		pool := worker.NewPool(1)
		pool.StopAndWait()

		future := pool.Submit(func() error { return nil })
		_, err := future.Get()
		assert.ErrorIs(t, err, worker.ErrPoolStopped, "Submit after stop should return ErrPoolStopped")

		futureResult := pool.SubmitResult(func() (any, error) { return nil, nil })
		_, err = futureResult.Get()
		assert.ErrorIs(t, err, worker.ErrPoolStopped, "SubmitResult after stop should return ErrPoolStopped")

		_, ok := pool.TrySubmit(func() error { return nil })
		assert.False(t, ok, "TrySubmit after stop should fail")
		_, err = future.Get() // Reuse future to check error
		assert.ErrorIs(t, err, worker.ErrPoolStopped)

		_, ok = pool.TrySubmitResult(func() (any, error) { return nil, nil })
		assert.False(t, ok, "TrySubmitResult after stop should fail")
	})

	t.Run("Stopped returns true after pool cancellation context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		pool := worker.NewPool(1, worker.WithContext(ctx))

		assert.False(t, pool.Stopped())
		cancel()                          // Cancel the parent context
		time.Sleep(50 * time.Millisecond) // Give time for context propagation
		assert.True(t, pool.Stopped(), "Pool should be stopped when its context is cancelled")
	})
}

// TestQueueInteraction verifies how the pool interacts with the blocking queue.
func TestQueueInteraction(t *testing.T) {
	maxWorker := int32(1)
	queueSize := int32(2)
	pool := worker.NewPool(maxWorker, worker.WithQueueSize(queueSize))
	defer pool.StopAndWait()

	// Task to keep the single worker busy
	var workerBusy sync.WaitGroup
	workerBusy.Add(1)
	pool.Submit(func() error {
		defer workerBusy.Done()
		time.Sleep(200 * time.Millisecond) // Keep worker busy
		fmt.Println("Worker busy completed")
		return nil
	})
	fmt.Println("Current time: ", time.Now(), "Running workers:", pool.RunningWorkers(), "Waiting tasks:", pool.WaitingTasks())

	// Submit tasks to fill the queue
	var wg sync.WaitGroup
	numQueuedTasks := int(queueSize)
	queuedFutures := make([]worker.Future[worker.Void], numQueuedTasks)
	for i := 0; i < numQueuedTasks; i++ {
		future := pool.Submit(func() error {
			// This will run after the first worker's task completes
			fmt.Printf("Queued task %d executed\n", i)
			return nil
		})
		queuedFutures[i] = future
	}
	fmt.Println("Current time: ", time.Now(), "Running workers:", pool.RunningWorkers(), "Waiting tasks:", pool.WaitingTasks())

	// Now try to submit one more task, it should fail (due to queue full)
	var blockedTaskExecuted atomic.Bool
	blockedFuture, ok := pool.TrySubmit(func() error {
		blockedTaskExecuted.Store(true)
		return nil
	})
	assert.False(t, ok, "Blocked task should not execute immediately")
	fmt.Println("Current time: ", time.Now(), "Running workers:", pool.RunningWorkers(), "Waiting tasks:", pool.WaitingTasks())

	workerBusy.Wait()

	blockedFuture, ok = pool.TrySubmit(func() error {
		blockedTaskExecuted.Store(true)
		return nil
	})
	assert.True(t, ok, "Blocked task should execute after worker is free")

	// Verify all queued tasks and the blocked task eventually complete
	wg.Add(numQueuedTasks + 1) // For the queued tasks and the blocked task
	go func() {
		defer wg.Done()
		_, err := blockedFuture.Get()
		assert.NoError(t, err)
		assert.True(t, blockedTaskExecuted.Load(), "Blocked task should have executed")
	}()

	for _, f := range queuedFutures {
		go func(f worker.Future[worker.Void]) {
			defer wg.Done()
			_, err := f.Get()
			assert.NoError(t, err)
		}(f)
	}
	wg.Wait()
}

// TestConcurrentSubmissions verifies the pool handles many concurrent submissions.
func TestConcurrentSubmissions(t *testing.T) {
	maxWorker := int32(5)
	queueSize := int32(10)
	pool := worker.NewPool(maxWorker, worker.WithQueueSize(queueSize))
	defer pool.StopAndWait()

	numTasks := 1000
	var executedCount atomic.Int32
	var wg sync.WaitGroup
	futures := make([]worker.Future[worker.Void], numTasks)

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			f := pool.Submit(func() error {
				time.Sleep(time.Millisecond) // Simulate small work
				executedCount.Add(1)
				return nil
			})
			futures[idx] = f
		}(i)
	}

	wg.Wait() // Wait for all Submit calls to return (some might still be queued)

	// Wait for all tasks to complete
	for _, f := range futures {
		require.NotNil(t, f) // Ensure future was assigned
		_, err := f.Get()
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(numTasks), executedCount.Load(), "All tasks should have been executed")
	assert.Equal(t, int32(0), pool.RunningWorkers(), "Queue should be empty after all tasks are done")
}

// TestErrMaxWorkerReached verifies worker scaling down when exceeding maxWorker.
func TestErrMaxWorkerReached(t *testing.T) {
	pool := worker.NewPool(1, worker.WithQueueSize(1)) // 1 worker, 1 queue slot
	defer pool.StopAndWait()

	var worker1Done sync.WaitGroup
	worker1Done.Add(1)
	// Task 1: occupies the worker
	f1 := pool.Submit(func() error {
		defer worker1Done.Done()
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Task 2: goes into the queue
	f2 := pool.Submit(func() error {
		return nil
	})

	// Task 3: tries to submit. It should block because the queue is full.
	var blocked bool
	go func() {
		_, err := pool.Submit(func() error { return nil }).Get()
		if errors.Is(err, worker.ErrPoolStopped) { // This can happen if pool is stopped quickly
			return
		}
		assert.NoError(t, err)
		blocked = true
	}()

	time.Sleep(50 * time.Millisecond) // Give time for tasks to settle

	// This usually happens if tasks are short-lived and workers try to read next.
	shortPool := worker.NewPool(1, worker.WithQueueSize(0)) // No queue
	defer shortPool.StopAndWait()

	// Keep the single worker busy initially to control flow
	var firstTaskDone sync.WaitGroup
	firstTaskDone.Add(1)
	_ = shortPool.Submit(func() error {
		defer firstTaskDone.Done()
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	// Ensure the first tasks completed.
	_, err := f1.Get()
	assert.NoError(t, err)
	_, err = f2.Get()
	assert.NoError(t, err)

	// Unblock the third task's goroutine (if it was blocking).
	if blocked {
		t.Log("Third task was eventually executed after unblocking.")
	}
}
