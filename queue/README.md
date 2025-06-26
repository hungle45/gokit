# BlockingQueue

`BlockingQueue` is a thread-safe, generic, and blocking queue implementation in Go. It provides both blocking and non-blocking methods for adding and retrieving elements, making it suitable for producer-consumer scenarios and other concurrent applications.

## Features

- **Generics:** Can store elements of any type.
- **Thread-Safe:** All operations are safe for concurrent access from multiple goroutines.
- **Bounded Capacity:** The queue can be configured with a maximum capacity to prevent uncontrolled growth.
- **Blocking Operations:**
    - `OfferWait(item T)`: Adds an item, waiting if the queue is full.
    - `GetWait() T`: Retrieves an item, waiting if the queue is empty.
- **Non-Blocking Operations:**
    - `Offer(item T) error`: Adds an item if space is available, otherwise returns `ErrQueueIsFull`.
    - `Get() (T, error)`: Retrieves an item if available, otherwise returns `ErrQueueIsEmpty`.
- **Iterator:** Provides a snapshot iterator over the elements in the queue.
- **JSON Marshaling:** Implements the `json.Marshaler` interface for easy serialization.

## Usage

### Creating a Queue

You can create a new `BlockingQueue` with a specific capacity and initial items.

```go
package main

import (
	"fmt"
	"github.com/hungle45/gokit/queue"
)

func main() {
	// Create a queue with a capacity of 5
	q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(5))

	// Create a queue with initial items and a capacity of 10
	initialItems := []string{"apple", "banana"}
	q2 := queue.NewBlockingQueue[string](initialItems, queue.WithCapacity(10))

	fmt.Printf("Queue 1 size: %d\n", q.Size())
	fmt.Printf("Queue 2 size: %d\n", q2.Size())
}
```

### Non-Blocking Operations (`Offer` and `Get`)

These methods return immediately with an error if the operation cannot be performed.

```go
package main

import (
	"fmt"
	"github.com/hungle45/gokit/queue"
)

func main() {
	q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(2))

	// Offer items
	err := q.Offer(1) // Success
	if err != nil {
		fmt.Println("Offer failed:", err)
	}
	err = q.Offer(2) // Success
	if err != nil {
		fmt.Println("Offer failed:", err)
	}
	err = q.Offer(3) // Fails, queue is full
	if err != nil {
		fmt.Println("Offer failed:", err) // Prints: Offer failed: queue is full
	}

	// Get items
	item, err := q.Get() // Success
	if err != nil {
		fmt.Println("Get failed:", err)
	} else {
		fmt.Println("Got item:", item) // Prints: Got item: 1
	}

	item, err = q.Get() // Success
	if err != nil {
		fmt.Println("Get failed:", err)
	} else {
		fmt.Println("Got item:", item) // Prints: Got item: 2
	}

	_, err = q.Get() // Fails, queue is empty
	if err != nil {
		fmt.Println("Get failed:", err) // Prints: Get failed: queue is empty
	}
}
```

### Blocking Operations (`OfferWait` and `GetWait`)

These methods are ideal for producer-consumer patterns where goroutines need to wait for the queue's state to change.

```go
package main

import (
	"fmt"
	"github.com/hungle45/gokit/queue"
	"sync"
	"time"
)

func main() {
	q := queue.NewBlockingQueue[int](nil, queue.WithCapacity(1))
	var wg sync.WaitGroup

	// Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Consumer: waiting for an item...")
		item := q.GetWait() // Blocks until an item is available
		fmt.Printf("Consumer: got item %d\n", item)
	}()

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Producer: preparing to offer an item...")
		time.Sleep(100 * time.Millisecond)
		q.OfferWait(42) // Offers an item, unblocking the consumer
		fmt.Println("Producer: item offered")
	}()

	wg.Wait()
}
```

### Iterating over the Queue

The `Iterator` method returns a channel that provides a snapshot of the queue's elements at the time of the call.

```go
package main

import (
	"fmt"
	"github.com/hungle45/gokit/queue"
)

func main() {
	initialItems := []string{"A", "B", "C"}
	q := queue.NewBlockingQueue[string](initialItems)

	fmt.Println("Iterating over queue snapshot:")
	for item := range q.Iterator() {
		fmt.Println("-", item)
	}
	
	// The original queue is not modified
	fmt.Printf("Queue size after iteration: %d\n", q.Size())
}
```

### JSON Serialization

The queue can be easily marshaled to JSON.

```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/hungle45/gokit/queue"
)

func main() {
	q := queue.NewBlockingQueue[int]([]int{1, 2, 3})
	
	jsonData, err := json.Marshal(q)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(jsonData)) // Prints: [1,2,3]
}
```