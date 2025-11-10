# conc (concurrency helpers)

The `conc` package provides small concurrency utilities used across the project. Its primary exported type is a lightweight `Group` abstraction that simplifies running a set of goroutines and waiting for their completion.

## Core concept

- `Group`: a small coordination helper that lets you launch goroutines and wait for all of them to finish. It's a tiny wrapper around `sync.WaitGroup` with a simple interface.

## API

- `type Group interface { Go(func()); Wait() }`
  - `Go(fn func())` launches `fn` in a new goroutine and tracks it.
  - `Wait()` blocks until all launched goroutines have completed.

- `func NewGroup() Group` — returns a new `Group` instance.

## Usage example

```go
package main

import (
    "fmt"
    "github.com/hungle45/gokit/conc"
)

func main() {
    g := conc.NewGroup()

    for i := 0; i < 3; i++ {
        idx := i
        g.Go(func() {
            // do work concurrently
            fmt.Println("worker", idx)
        })
    }

    // Wait for all workers to finish
    g.Wait()
}
```

## Notes

- This package intentionally provides a minimal, test-friendly contract. It is safe to call `Go` from multiple goroutines.
- If you need richer features (error propagation, cancellation, limits), consider using higher-level constructs or packages built on top of this `Group`.
