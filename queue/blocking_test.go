package queue_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hungle45/gokit/queue"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewBlockingQueue tests the NewBlockingQueue constructor with various options.
func TestNewBlockingQueue(t *testing.T) {
	t.Run("Default capacity (MaxInt)", func(t *testing.T) {
		q := queue.NewBlockingQueue[int]([]int{})
		assert.True(t, q.IsEmpty(), "Queue should be empty initially")
		assert.Equal(t, 0, q.Size(), "Initial size should be 0")
		// Cannot directly assert MaxInt, but ensure it doesn't immediately become full
		for i := 0; i < 1000; i++ {
			err := q.Offer(i)
			assert.NoError(t, err, "Should be able to offer to a default capacity queue")
		}
	})

	t.Run("With specified capacity", func(t *testing.T) {
		capacity := 5
		q := queue.NewBlockingQueue[string](nil, queue.WithCapacity(capacity))
		assert.True(t, q.IsEmpty(), "Queue should be empty initially")
		assert.Equal(t, 0, q.Size(), "Initial size should be 0")

		for i := 0; i < capacity; i++ {
			err := q.Offer(fmt.Sprintf("item-%d", i))
			assert.NoError(t, err, "Offer should succeed within capacity")
		}
		assert.Equal(t, capacity, q.Size(), "Size should match capacity after filling")

		err := q.Offer("extra-item")
		assert.ErrorIs(t, err, queue.ErrQueueIsFull, "Offer should fail when queue is full")
		assert.Equal(t, capacity, q.Size(), "Size should remain capacity when full")
	})

	t.Run("With initial items respecting capacity", func(t *testing.T) {
		initialItems := []int{10, 20, 30, 40, 50}
		capacity := 3
		q := queue.NewBlockingQueue[int](initialItems, queue.WithCapacity(capacity))
		assert.False(t, q.IsEmpty(), "Queue should not be empty")
		assert.Equal(t, capacity, q.Size(), "Size should be limited by capacity, not initial items length")

		expectedItems := []int{10, 20, 30}
		for _, expected := range expectedItems {
			item, err := q.Get()
			assert.NoError(t, err)
			assert.Equal(t, expected, item)
		}
		assert.True(t, q.IsEmpty())
	})
}

// TestOfferAndGetNonBlocking tests the non-blocking Offer and Get methods.
func TestOfferAndGetNonBlocking(t *testing.T) {
	q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(2))

	t.Run("Offer to empty queue", func(t *testing.T) {
		err := q.Offer(1)
		assert.NoError(t, err, "Offer should succeed")
		assert.Equal(t, 1, q.Size(), "Size should be 1")
		assert.False(t, q.IsEmpty(), "Queue should not be empty")
	})

	t.Run("Offer to non-empty queue", func(t *testing.T) {
		err := q.Offer(2)
		assert.NoError(t, err, "Offer should succeed")
		assert.Equal(t, 2, q.Size(), "Size should be 2")
	})

	t.Run("Offer to full queue", func(t *testing.T) {
		err := q.Offer(3)
		assert.ErrorIs(t, err, queue.ErrQueueIsFull, "Offer should return ErrQueueIsFull")
		assert.Equal(t, 2, q.Size(), "Size should remain 2")
	})

	t.Run("Get from non-empty queue", func(t *testing.T) {
		val, err := q.Get()
		assert.NoError(t, err, "Get should succeed")
		assert.Equal(t, 1, val, "Should get the first element offered")
		assert.Equal(t, 1, q.Size(), "Size should be 1")
	})

	t.Run("Get from non-empty queue again", func(t *testing.T) {
		val, err := q.Get()
		assert.NoError(t, err, "Get should succeed")
		assert.Equal(t, 2, val, "Should get the next element")
		assert.Equal(t, 0, q.Size(), "Size should be 0")
		assert.True(t, q.IsEmpty(), "Queue should be empty")
	})

	t.Run("Get from empty queue", func(t *testing.T) {
		_, err := q.Get()
		assert.ErrorIs(t, err, queue.ErrQueueIsEmpty, "Get should return ErrQueueIsEmpty")
		assert.Equal(t, 0, q.Size(), "Size should remain 0")
	})
}

// TestOfferWaitAndGetWaitBlocking tests the blocking OfferWait and GetWait methods.
func TestOfferWaitAndGetWaitBlocking(t *testing.T) {
	q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(1))

	t.Run("GetWait blocks until OfferWait unblocks", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2) // One for getter, one for setter

		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Give GetWait a chance to block
			q.OfferWait(100)
		}()

		var received int
		go func() {
			defer wg.Done()
			received = q.GetWait()
		}()

		// Use a channel to detect if GetWait returns immediately (failure)
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			assert.Equal(t, 100, received, "GetWait should receive the correct value")
			assert.Equal(t, 0, q.Size(), "Queue should be empty after GetWait")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Test timed out, GetWait or OfferWait did not unblock")
		}
	})

	t.Run("OfferWait blocks until GetWait unblocks", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(1))
		assert.NoError(t, q.Offer(1)) // Fill the queue

		var wg sync.WaitGroup
		wg.Add(2) // One for setter, one for getter

		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Give OfferWait a chance to block
			val, err := q.Get()
			assert.NoError(t, err)
			assert.Equal(t, 1, val)
		}()

		var offerWaitFinished bool
		go func() {
			defer wg.Done()
			q.OfferWait(2) // This should block
			offerWaitFinished = true
		}()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			assert.True(t, offerWaitFinished, "OfferWait should have completed")
			assert.Equal(t, 1, q.Size(), "Queue should contain the newly offered item")
			val, err := q.Get()
			assert.NoError(t, err)
			assert.Equal(t, 2, val)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Test timed out, OfferWait or Get did not unblock")
		}
	})

	t.Run("Multiple concurrent producers and consumers", func(t *testing.T) {
		capacity := 5
		numProducers := 3
		numConsumers := 3
		itemsPerProducer := 100
		totalItems := numProducers * itemsPerProducer

		q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(capacity))
		var wg sync.WaitGroup
		produced := make(chan int, totalItems)
		consumed := make(chan int, totalItems)

		// Producers
		for i := 0; i < numProducers; i++ {
			wg.Add(1)
			go func(producerID int) {
				defer wg.Done()
				for j := 0; j < itemsPerProducer; j++ {
					item := producerID*itemsPerProducer + j
					q.OfferWait(item)
					produced <- item
				}
			}(i)
		}

		// Consumers
		for i := 0; i < numConsumers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < itemsPerProducer; j++ { // Each consumer should get `itemsPerProducer` to consume all
					// This loop condition needs to be totalItems / numConsumers if all consumers get roughly equal share
					// Or just loop until the queue is exhausted or a specific number of items are received
					// For simplicity, we'll just have them try to get items, and collect them.
					// The main check will be on the final consumed slice.
					select {
					case <-time.After(2 * time.Second): // Timeout for consumer to prevent hang
						return
					default:
						val := q.GetWait()
						consumed <- val
					}
				}
			}()
		}

		// Wait for producers to finish
		producerDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(producerDone)
			close(produced) // Close produced channel when all producers are done
		}()

		// Collect produced items
		var actualProduced []int
		for item := range produced {
			actualProduced = append(actualProduced, item)
		}

		// Allow consumers to finish after producers
		var wgConsumers sync.WaitGroup
		wgConsumers.Add(numConsumers)
		for i := 0; i < numConsumers; i++ {
			go func() {
				defer wgConsumers.Done()
				// Keep consuming until no more items or timeout
				for q.Size() > 0 || len(actualProduced) > len(consumed) { // Heuristic: keep consuming while items are expected
					select {
					case <-producerDone: // Producers are done, check if queue is truly empty
						if q.Size() == 0 {
							return
						}
					case <-time.After(10 * time.Millisecond): // Small delay to avoid busy waiting
					}
					if val, err := q.Get(); err == nil {
						consumed <- val
					} else if errors.Is(err, queue.ErrQueueIsEmpty) {
						// This is okay, just means the queue is temporarily empty, might get more from other producers
						time.Sleep(1 * time.Millisecond) // Give other goroutines a chance
					}
				}
			}()
		}

		// Use Eventually to check if all items are consumed
		// The `consumed` channel might not be closed naturally in all paths,
		// so we rely on the final check after a reasonable delay.
		var finalConsumed []int
		timeout := time.After(3 * time.Second) // Increased timeout for potentially slow concurrent operations

	ConsumptionLoop:
		for len(finalConsumed) < totalItems {
			select {
			case val, ok := <-consumed:
				if ok {
					finalConsumed = append(finalConsumed, val)
				} else {
					// Channel closed, but not all items consumed, might indicate an issue or race
					// Break and let the final assert handle it.
					break ConsumptionLoop
				}
			case <-timeout:
				t.Fatalf("Timeout waiting for all items to be consumed. Expected %d, Got %d. Queue size: %d", totalItems, len(finalConsumed), q.Size())
			case <-producerDone:
				if q.Size() == 0 && len(finalConsumed) == totalItems {
					break ConsumptionLoop
				}
			}
		}

		// Final check after all operations should be done
		assert.Equal(t, totalItems, len(finalConsumed), "All items should be consumed")
		assert.Equal(t, 0, q.Size(), "Queue should be empty at the end")

		// Check if all original items are present in the consumed list (order might differ)
		producedMap := make(map[int]int)
		for _, item := range actualProduced {
			producedMap[item]++
		}
		consumedMap := make(map[int]int)
		for _, item := range finalConsumed {
			consumedMap[item]++
		}
		assert.Equal(t, producedMap, consumedMap, "Consumed items should match produced items")
	})
}

// TestPeekAndPeekWait tests the Peek and PeekWait methods.
func TestPeekAndPeekWait(t *testing.T) {
	q := queue.NewBlockingQueue[string](nil, queue.WithCapacity(2))

	t.Run("Peek on empty queue", func(t *testing.T) {
		_, err := q.Peek()
		assert.ErrorIs(t, err, queue.ErrQueueIsEmpty, "Peek should return ErrQueueIsEmpty")
	})

	t.Run("Peek after offering", func(t *testing.T) {
		assert.NoError(t, q.Offer("first"), "First offer should succeed")
		assert.NoError(t, q.Offer("second"), "Second offer should succeed")

		val, err := q.Peek()
		assert.NoError(t, err, "Peek should succeed")
		assert.Equal(t, "first", val, "Peek should return the head without removing it")
		assert.Equal(t, 2, q.Size(), "Size should remain unchanged after Peek")

		// Verify it's still the head after another Get
		getVal, _ := q.Get()
		assert.Equal(t, "first", getVal, "First element should still be removable after Peek")
	})

	t.Run("PeekWait blocks until element is available", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(1)) // New queue for this sub-test
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Give PeekWait a chance to block
			q.OfferWait(123)
		}()

		var peekedValue int
		go func() {
			defer wg.Done()
			peekedValue = q.PeekWait()
		}()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			assert.Equal(t, 123, peekedValue, "PeekWait should receive the correct value")
			assert.Equal(t, 1, q.Size(), "Queue size should be 1 after PeekWait (element not removed)")
			// Verify it's still there
			val, _ := q.Get()
			assert.Equal(t, 123, val)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Test timed out, PeekWait did not unblock")
		}
	})
}

// TestSizeAndIsEmpty tests the Size and IsEmpty methods.
func TestSizeAndIsEmpty(t *testing.T) {
	q := queue.NewBlockingQueue[float64](nil, queue.WithCapacity(3))

	assert.True(t, q.IsEmpty(), "Queue should be empty initially")
	assert.Equal(t, 0, q.Size(), "Initial size should be 0")

	assert.NoError(t, q.Offer(1.1))
	assert.False(t, q.IsEmpty(), "Queue should not be empty after offer")
	assert.Equal(t, 1, q.Size(), "Size should be 1 after one offer")

	assert.NoError(t, q.Offer(2.2))
	assert.Equal(t, 2, q.Size(), "Size should be 2 after two offers")

	_, err := q.Get()
	assert.NoError(t, err)
	assert.False(t, q.IsEmpty(), "Queue should not be empty after get if elements remain")
	assert.Equal(t, 1, q.Size(), "Size should be 1 after one get")

	_, err = q.Get()
	assert.NoError(t, err)
	assert.True(t, q.IsEmpty(), "Queue should be empty after all elements are retrieved")
	assert.Equal(t, 0, q.Size(), "Size should be 0 after all elements are retrieved")
}

// TestClear tests the Clear method.
func TestClear(t *testing.T) {
	t.Run("Clear non-empty queue", func(t *testing.T) {
		initialItems := []int{1, 2, 3, 4, 5}
		q := queue.NewBlockingQueue[int](initialItems)

		assert.Equal(t, 5, q.Size(), "Queue should have 5 items initially")

		removedItems := q.Clear()
		assert.True(t, q.IsEmpty(), "Queue should be empty after Clear")
		assert.Equal(t, 0, q.Size(), "Size should be 0 after Clear")
		assert.ElementsMatch(t, initialItems, removedItems, "Removed items should match initial items")
	})

	t.Run("Clear empty queue", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil)
		removedItems := q.Clear()
		assert.True(t, q.IsEmpty(), "Queue should still be empty")
		assert.Equal(t, 0, q.Size(), "Size should still be 0")
		assert.Nil(t, removedItems, "Removed items should be nil for an empty queue")
	})

	t.Run("Clear unblocks waiting OfferWait", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(1))
		assert.NoError(t, q.Offer(10)) // Fill the queue

		var wg sync.WaitGroup
		wg.Add(1)

		var offerFinished bool
		go func() {
			defer wg.Done()
			q.OfferWait(20) // This should block until Clear is called
			offerFinished = true
		}()

		time.Sleep(50 * time.Millisecond) // Give OfferWait a chance to block

		assert.Equal(t, 1, q.Size(), "Queue size should be 1 before clear")
		q.Clear() // This should unblock OfferWait

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			assert.True(t, offerFinished, "OfferWait should have unblocked and finished")
			assert.Equal(t, 1, q.Size(), "Queue should contain the item offered after Clear")
			val, _ := q.Get()
			assert.Equal(t, 20, val)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Test timed out, OfferWait was not unblocked by Clear")
		}
	})
}

// TestIterator tests the Iterator method.
func TestIterator(t *testing.T) {
	t.Run("Iterate over non-empty queue", func(t *testing.T) {
		initialItems := []string{"apple", "banana", "cherry"}
		q := queue.NewBlockingQueue[string](initialItems)

		var receivedItems []string
		for item := range q.Iterator() {
			receivedItems = append(receivedItems, item)
		}
		assert.ElementsMatch(t, initialItems, receivedItems, "Iterator should yield all current elements")
		assert.Equal(t, len(initialItems), q.Size(), "Queue size should remain unchanged after iteration")
	})

	t.Run("Iterate over empty queue", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil)
		var receivedItems []int
		for item := range q.Iterator() {
			receivedItems = append(receivedItems, item)
		}
		assert.Empty(t, receivedItems, "Iterator should yield no elements for an empty queue")
	})

	t.Run("Iterator provides a snapshot", func(t *testing.T) {
		q := queue.NewBlockingQueue[int]([]int{1, 2}, queue.WithCapacity(5))
		iteratorCh := q.Iterator()

		assert.NoError(t, q.Offer(3)) // Add an item after iterator is created
		_, err := q.Get()             // Remove an item after iterator is created
		assert.NoError(t, err)

		var receivedItems []int
		for item := range iteratorCh {
			receivedItems = append(receivedItems, item)
		}
		assert.ElementsMatch(t, []int{1, 2}, receivedItems, "Iterator should reflect the queue state at creation time")
		assert.Equal(t, 2, q.Size(), "Queue should now have 2 items (1, 3)") // Verify actual queue state
	})
}

// TestMarshalJSON tests the MarshalJSON method.
func TestMarshalJSON(t *testing.T) {
	t.Run("Marshal non-empty queue of integers", func(t *testing.T) {
		q := queue.NewBlockingQueue[int]([]int{10, 20, 30})
		jsonData, err := q.MarshalJSON()
		assert.NoError(t, err, "MarshalJSON should not return an error")

		expectedJSON := `[10,20,30]`
		assert.JSONEq(t, expectedJSON, string(jsonData), "JSON output should match expected for integers")
	})

	t.Run("Marshal non-empty queue of strings", func(t *testing.T) {
		q := queue.NewBlockingQueue[string]([]string{"alpha", "beta"})
		jsonData, err := q.MarshalJSON()
		assert.NoError(t, err, "MarshalJSON should not return an error")

		expectedJSON := `["alpha","beta"]`
		assert.JSONEq(t, expectedJSON, string(jsonData), "JSON output should match expected for strings")
	})

	t.Run("Marshal empty queue", func(t *testing.T) {
		q := queue.NewBlockingQueue[int](nil)
		jsonData, err := q.MarshalJSON()
		assert.NoError(t, err, "MarshalJSON should not return an error")

		expectedJSON := `[]`
		assert.JSONEq(t, expectedJSON, string(jsonData), "JSON output should be an empty array for an empty queue")
	})

	t.Run("Unmarshal and verify type", func(t *testing.T) {
		q := queue.NewBlockingQueue[string]([]string{"one", "two"})
		jsonData, err := q.MarshalJSON()
		assert.NoError(t, err)

		var result []string
		err = json.Unmarshal(jsonData, &result)
		assert.NoError(t, err, "Should be able to unmarshal the JSON data")
		assert.ElementsMatch(t, []string{"one", "two"}, result, "Unmarshal data should match original elements")
	})
}
