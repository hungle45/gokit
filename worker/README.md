# Worker Pool

The `worker` package provides a powerful and flexible worker pool implementation for managing and executing concurrent tasks in Go. It includes a `Future`-based API for handling asynchronous results, errors, and cancellations.

## Core Components

- **`Pool`**: A thread-safe pool of workers that manages a queue of tasks. It can dynamically adjust the number of active workers up to a configured maximum.
- **`Future`**: Represents the result of an asynchronous computation. It allows you to retrieve the result (or error) of a task once it completes, with optional timeouts.

---

## `Future`

A `Future` is a placeholder for a value that will be available later. It's returned when you submit a task to the pool.

### Key Methods

- `Get() (T, error)`: Waits indefinitely for the task to complete and returns its result or an error.
- `GetWithTimeOut(timeout time.Duration) (T, error)`: Waits for the task to complete for a specified duration. If the timeout is reached, it returns `worker.ErrTimeout`.
- `Done() <-chan struct{}`: Returns a channel that is closed when the task is complete.
- `Cancel()`: Attempts to cancel the task. If successful, `Get()` will return `worker.ErrTaskCancelled`.

### Example: Using a `FutureTask`

While you typically get a `Future` from the pool, you can create a `FutureTask` directly.

```go
package main

import (
	"context"
	"fmt"
	"github.com/hungle45/gokit/worker"
	"time"
)

func main() {
	// Create a callable function
	callable := func() (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "Hello, Future!", nil
	}

	// Create a new FutureTask
	ft := worker.NewFutureTask[string](context.Background(), callable)

	// Run the task in a separate goroutine
	go ft.Run()

	fmt.Println("Waiting for the result...")

	// Get the result
	result, err := ft.Get()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Result:", result) // Prints: Result: Hello, Future!
}
```

---

## `Pool`

The `Pool` manages a collection of goroutines (workers) to execute tasks concurrently.

### Creating a Pool

You create a pool by specifying the maximum number of workers. You can also provide options to configure the task queue size and a parent context for cancellation.

```go
package main

import (
	"github.com/hungle45/gokit/worker"
)

func main() {
	// Create a pool with up to 10 workers and a task queue of size 50.
	pool := worker.NewPool(10, worker.WithQueueSize(50))

	// It's important to stop the pool when you're done to release resources.
	defer pool.StopAndWait()

	// ... use the pool
}
```

### Submitting Tasks

There are two main ways to submit tasks:

1.  **`Submit(func() error)`**: For tasks that don't return a value (fire-and-forget). It returns a `Future[Void]`.
2.  **`SubmitResult(func() (any, error))`**: For tasks that return a value. It returns a `Future[any]`.

Both methods have non-blocking `TrySubmit` variants that return `false` if the pool's queue is full.

### Example: Full Worker Pool Usage

This example demonstrates creating a pool, submitting various tasks, and handling their results.

```go
package main

import (
	"fmt"
	"github.com/hungle45/gokit/worker"
	"time"
)

func main() {
	// Create a pool with a max of 3 workers and a queue size of 5.
	pool := worker.NewPool(3, worker.WithQueueSize(5))
	defer pool.StopAndWait()

	// --- Task 1: A simple function that returns no result ---
	fmt.Println("Submitting a simple function...")
	future1 := pool.Submit(func() error {
		fmt.Println("Task 1: Executing...")
		time.Sleep(50 * time.Millisecond)
		fmt.Println("Task 1: Done.")
		return nil
	})

	// --- Task 2: A function that returns a result ---
	fmt.Println("Submitting a function with a result...")
	future2 := pool.SubmitResult(func() (any, error) {
		fmt.Println("Task 2: Calculating...")
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})

	// --- Task 3: A function that returns an error ---
	fmt.Println("Submitting a function that will fail...")
	future3 := pool.SubmitResult(func() (any, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	// --- Wait for results ---
	fmt.Println("Waiting for Task 1 (no result)...")
	_, err1 := future1.Get()
	if err1 != nil {
		fmt.Printf("Task 1 failed: %v\n", err1)
	} else {
		fmt.Println("Task 1 completed successfully.")
	}

	fmt.Println("Waiting for Task 2 (result)...")
	result2, err2 := future2.Get()
	if err2 != nil {
		fmt.Printf("Task 2 failed: %v\n", err2)
	} else {
		fmt.Printf("Task 2 result: %d\n", result2.(int))
	}

	fmt.Println("Waiting for Task 3 (error)...")
	_, err3 := future3.Get()
	if err3 != nil {
		fmt.Printf("Task 3 failed as expected: %v\n", err3)
	}
}
```

### Shutting Down the Pool

- **`StopAndWait()`**: Blocks until all currently running and queued tasks are completed, then stops the pool. No new tasks can be submitted after calling this.
- **`Stop() Future[Void]`**: Initiates a graceful shutdown non-blockingly. It returns a `Future` that completes when the shutdown is finished.

## Error Handling

The package defines several standard errors:
- `ErrPoolStopped`: Returned when a task is submitted to a stopped pool.
- `ErrPoolFull`: Returned by `TrySubmit` methods when the worker and queue capacity is exhausted.
- `ErrTimeout`: Returned by `Future.GetWithTimeOut` on timeout.
- `ErrTaskCancelled`: Returned by `Future.Get` if the task was cancelled.
- `ErrPanic`: Returned by `Future.Get` if the task panicked during execution.
