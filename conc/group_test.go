package conc

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		delay time.Duration
	}{
		{name: "no goroutines", n: 0, delay: 0},
		{name: "single", n: 1, delay: 10 * time.Millisecond},
		{name: "multiple", n: 5, delay: 5 * time.Millisecond},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := NewGroup()
			var counter int32

			for i := 0; i < tc.n; i++ {
				g.Go(func() {
					if tc.delay > 0 {
						time.Sleep(tc.delay)
					}
					atomic.AddInt32(&counter, 1)
				})
			}

			// ensure Wait returns within a reasonable timeout
			done := make(chan struct{})
			go func() {
				g.Wait()
				close(done)
			}()

			select {
			case <-done:
				// success
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("Wait() timed out for case %q (n=%d)", tc.name, tc.n)
			}

			if got := int(atomic.LoadInt32(&counter)); got != tc.n {
				t.Fatalf("counter = %d, want %d", got, tc.n)
			}
		})
	}
}
