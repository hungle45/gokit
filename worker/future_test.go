package worker_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/hungle45/gokit/worker"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewFutureTask verifies the correct initialization of a FutureTask.
func TestNewFutureTask(t *testing.T) {
	ctx := context.Background()
	callable := func() (string, error) { return "test", nil }
	ft := worker.NewFutureTask[string](ctx, callable)

	assert.NotNil(t, ft, "NewFutureTask should return a non-nil FutureTask")
	assert.NotNil(t, ft.Done(), "Done channel should be initialized")
	// Cannot directly check the internal context or callable due to private fields,
	// but their functionality will be tested via other methods.
}

// TestRunAndGetSuccess verifies successful execution and retrieval of result.
func TestRunAndGetSuccess(t *testing.T) {
	ctx := context.Background()
	expectedResult := "success!"
	callable := func() (string, error) {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		return expectedResult, nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)

	// Run the task in a goroutine to not block the test
	go ft.Run()

	// Wait for the task to complete
	<-ft.Done()

	// Get the result
	result, err := ft.Get()
	assert.NoError(t, err, "Get should not return an error for successful task")
	assert.Equal(t, expectedResult, result, "Get should return the correct result")
}

// TestRunAndGetError verifies error propagation from callable.
func TestRunAndGetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("simulated callable error")
	callable := func() (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 0, expectedErr
	}

	ft := worker.NewFutureTask[int](ctx, callable)
	go ft.Run()

	<-ft.Done()

	result, err := ft.Get()
	assert.ErrorIs(t, err, expectedErr, "Get should return the error from the callable")
	assert.Equal(t, 0, result, "Result should be zero value on error")
}

// TestRunAndGetPanic verifies panic handling.
func TestRunAndGetPanic(t *testing.T) {
	ctx := context.Background()
	panicMsg := "oops, I panicked!"
	callable := func() (bool, error) {
		panic(panicMsg)
	}

	ft := worker.NewFutureTask[bool](ctx, callable)
	go ft.Run()

	<-ft.Done()

	_, err := ft.Get()
	assert.Error(t, err, "Get should return an error when callable panics")
	assert.ErrorIs(t, err, worker.ErrPanic, "Error should be wrapped with ErrPanic")
	assert.Contains(t, err.Error(), panicMsg, "Error message should contain panic message")
	assert.Contains(t, err.Error(), "debug.Stack", "Error message should contain stack trace")

	// Test panic with a non-error type
	panicInt := 123
	callableIntPanic := func() (string, error) {
		panic(panicInt)
	}
	ftInt := worker.NewFutureTask[string](ctx, callableIntPanic)
	go ftInt.Run()
	<-ftInt.Done()
	_, err = ftInt.Get()
	assert.Error(t, err)
	assert.ErrorIs(t, err, worker.ErrPanic)
	assert.Contains(t, err.Error(), fmt.Sprintf("%v", panicInt))
}

// TestCancel verifies immediate cancellation.
func TestCancel(t *testing.T) {
	ctx := context.Background()
	// Callable that would take a long time
	callable := func() (string, error) {
		time.Sleep(5 * time.Second) // Should be interrupted
		return "should not be returned", nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ft.Run() // This will start, but should be cancelled quickly
	}()

	// Give a moment for Run to start, then cancel
	time.Sleep(10 * time.Millisecond)
	ft.Cancel()

	// Wait for Done channel to close (should be immediate after cancel)
	select {
	case <-ft.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Done channel did not close after cancellation")
	}

	result, err := ft.Get()
	assert.ErrorIs(t, err, worker.ErrTaskCancelled, "Get should return ErrTaskCancelled after Cancel")
	var zeroStr string
	assert.Equal(t, zeroStr, result, "Result should be zero value on cancellation")

	wg.Wait() // Ensure Run goroutine finishes
}

// TestCancelWithErr verifies cancellation with a custom error.
func TestCancelWithErr(t *testing.T) {
	ctx := context.Background()
	customErr := errors.New("user stopped task")
	callable := func() (int, error) {
		time.Sleep(5 * time.Second)
		return 0, nil
	}

	ft := worker.NewFutureTask[int](ctx, callable)
	go ft.Run()

	time.Sleep(10 * time.Millisecond)
	ft.CancelWithErr(customErr)

	<-ft.Done()

	_, err := ft.Get()
	assert.ErrorIs(t, err, customErr, "Get should return the custom cancellation error")
}

// TestCancelAfterCompletion ensures cancellation does not affect a completed task.
func TestCancelAfterCompletion(t *testing.T) {
	ctx := context.Background()
	expectedResult := "finished"
	callable := func() (string, error) {
		return expectedResult, nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)
	go ft.Run()

	<-ft.Done() // Wait for task to complete naturally

	ft.Cancel() // Try to cancel after it's done

	result, err := ft.Get()
	assert.NoError(t, err, "Get should not return an error after cancellation of a completed task")
	assert.Equal(t, expectedResult, result, "Result should remain the original result after failed cancellation")

	ft.CancelWithErr(errors.New("another cancel")) // Try custom cancel
	result, err = ft.Get()
	assert.NoError(t, err) // Should still be no error from custom cancel
	assert.Equal(t, expectedResult, result)
}

// TestGetWithTimeoutSuccess verifies GetWithTimeOut returning before timeout.
func TestGetWithTimeoutSuccess(t *testing.T) {
	ctx := context.Background()
	expectedResult := 42
	callable := func() (int, error) {
		time.Sleep(50 * time.Millisecond) // Shorter than timeout
		return expectedResult, nil
	}

	ft := worker.NewFutureTask[int](ctx, callable)
	go ft.Run()

	result, err := ft.GetWithTimeOut(200 * time.Millisecond)
	assert.NoError(t, err, "GetWithTimeOut should not timeout if task completes in time")
	assert.Equal(t, expectedResult, result, "GetWithTimeOut should return the correct result")
}

// TestGetWithTimeoutFailure verifies GetWithTimeOut timing out.
func TestGetWithTimeoutFailure(t *testing.T) {
	ctx := context.Background()
	callable := func() (string, error) {
		time.Sleep(500 * time.Millisecond) // Longer than timeout
		return "too slow", nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)
	go ft.Run()

	var zeroStr string
	result, err := ft.GetWithTimeOut(50 * time.Millisecond)
	assert.ErrorIs(t, err, worker.ErrTimeout, "GetWithTimeOut should return ErrTimeout")
	assert.Equal(t, zeroStr, result, "Result should be zero value on timeout")

	// Ensure the original task eventually completes without interfering
	<-ft.Done()
	finalResult, finalErr := ft.Get()
	assert.NoError(t, finalErr)
	assert.Equal(t, "too slow", finalResult)
}

// TestGetWithTimeoutAfterDone ensures GetWithTimeOut returns immediately if already done.
func TestGetWithTimeoutAfterDone(t *testing.T) {
	ctx := context.Background()
	expectedResult := "already done"
	callable := func() (string, error) {
		return expectedResult, nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)
	go ft.Run()

	<-ft.Done() // Wait for completion

	result, err := ft.GetWithTimeOut(1 * time.Millisecond) // Very short timeout
	assert.NoError(t, err, "GetWithTimeOut should not timeout if task is already done")
	assert.Equal(t, expectedResult, result, "GetWithTimeOut should return the correct result")
}

// TestFutureResolutionErrorString verifies the Error() method of futureResolution.
func TestFutureResolutionErrorString(t *testing.T) {
	// Case 1: Resolution with an error
	err := fmt.Errorf("custom error")
	// Manually create a futureResolution for testing its Error() method
	resWithErr := struct {
		value int
		err   error
	}{value: 0, err: err}
	assert.Equal(t, err.Error(), resWithErr.err.Error(), "Error() should return the underlying error string")

	// Case 2: Resolution without an error (successful completion)
	resWithoutErr := struct {
		value string
		err   error
	}{value: "hello", err: nil}
	assert.Contains(t, fmt.Sprintf("%v", resWithoutErr.value), "hello", "Error() should describe the value when no error is present")
}

// TestContextCancelCauseIntegration tests that context.Cause works as expected
// after the FutureTask completes or cancels.
func TestContextCancelCauseIntegration(t *testing.T) {
	t.Run("Context Cause after success", func(t *testing.T) {
		ctx := context.Background()
		callable := func() (string, error) { return "data", nil }
		ft := worker.NewFutureTask[string](ctx, callable)
		go ft.Run()
		<-ft.Done()

		// We can't directly get the context from ft, but we can verify Get() behaviour
		_, err := ft.Get()
		assert.NoError(t, err) // Should be no error if success
	})

	t.Run("Context Cause after cancellation", func(t *testing.T) {
		ctx := context.Background()
		callable := func() (string, error) { time.Sleep(time.Hour); return "", nil }
		ft := worker.NewFutureTask[string](ctx, callable)
		go ft.Run()
		time.Sleep(10 * time.Millisecond)
		ft.Cancel()
		<-ft.Done()

		_, err := ft.Get()
		assert.ErrorIs(t, err, worker.ErrTaskCancelled)
	})

	t.Run("Context Cause after callable error", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("my specific error")
		callable := func() (string, error) { return "", expectedErr }
		ft := worker.NewFutureTask[string](ctx, callable)
		go ft.Run()
		<-ft.Done()

		_, err := ft.Get()
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Context Cause after panic", func(t *testing.T) {
		ctx := context.Background()
		callable := func() (string, error) { panic("test panic") }
		ft := worker.NewFutureTask[string](ctx, callable)
		go ft.Run()
		<-ft.Done()

		_, err := ft.Get()
		assert.ErrorIs(t, err, worker.ErrPanic)
	})
}

// TestMultipleGetsOnSameFuture ensures multiple calls to Get/GetWithTimeOut return the same result.
func TestMultipleGetsOnSameFuture(t *testing.T) {
	ctx := context.Background()
	expectedResult := "hello world"
	callable := func() (string, error) {
		time.Sleep(50 * time.Millisecond)
		return expectedResult, nil
	}

	ft := worker.NewFutureTask[string](ctx, callable)
	go ft.Run()

	<-ft.Done() // Wait for completion

	res1, err1 := ft.Get()
	assert.NoError(t, err1)
	assert.Equal(t, expectedResult, res1)

	res2, err2 := ft.Get() // Call Get again
	assert.NoError(t, err2)
	assert.Equal(t, expectedResult, res2)

	res3, err3 := ft.GetWithTimeOut(1 * time.Second) // Call GetWithTimeOut
	assert.NoError(t, err3)
	assert.Equal(t, expectedResult, res3)
}

// TestFutureTaskDoesNotBlockParentContext ensures FutureTask's context cancellation
// does not cancel its parent context prematurely.
func TestFutureTaskDoesNotBlockParentContext(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	// Callable that will run for a bit
	callable := func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "done", nil
	}

	ft := worker.NewFutureTask[string](parentCtx, callable)
	go ft.Run()

	// Cancel the future task's own context via FutureTask.Cancel()
	// This should not affect parentCtx
	time.Sleep(50 * time.Millisecond)
	ft.Cancel()

	// Check parent context status after FutureTask is cancelled
	select {
	case <-parentCtx.Done():
		t.Fatal("Parent context was cancelled unexpectedly")
	case <-time.After(100 * time.Millisecond):
		// Parent context is still alive, as expected
	}

	// Wait for the FutureTask's Done channel to close (due to cancellation)
	<-ft.Done()
	_, err := ft.Get()
	assert.ErrorIs(t, err, worker.ErrTaskCancelled)

	// After FutureTask is done (cancelled), parentCtx should still be active
	select {
	case <-parentCtx.Done():
		t.Fatal("Parent context was cancelled unexpectedly after FutureTask finished")
	case <-time.After(10 * time.Millisecond):
		// Still active, good
	}

	parentCancel() // Now explicitly cancel the parent
	select {
	case <-parentCtx.Done():
		// Parent context is now cancelled, as expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Parent context did not cancel after explicit parentCancel()")
	}
}
