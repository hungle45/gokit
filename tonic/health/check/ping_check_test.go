package check

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPPingCheckerOpenAndClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	host, port, _ := net.SplitHostPort(ln.Addr().String())

	// accept connections and close immediately
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	t.Run("open", func(t *testing.T) {
		c := NewTCPPingChecker(host, port)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		ss := c.Check(ctx)
		if ss.Status != StatusUp {
			t.Fatalf("expected UP for open port, got %v", ss.Status)
		}
		if addr, ok := ss.Details["address"]; !ok || addr == "" {
			t.Fatalf("expected address detail for open, got %v", ss.Details)
		}
		if _, ok := ss.Details["time_to_connect_ms"]; !ok {
			t.Fatalf("expected time_to_connect_ms for open, got %v", ss.Details)
		}
	})

	t.Run("closed", func(t *testing.T) {
		c := NewTCPPingChecker("127.0.0.1", "1")
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		ss := c.Check(ctx)
		if ss.Status != StatusDown {
			t.Fatalf("expected DOWN for closed port, got %v", ss.Status)
		}
		if addr, ok := ss.Details["address"]; !ok || addr == "" {
			t.Fatalf("expected address detail for closed, got %v", ss.Details)
		}
		if _, ok := ss.Details["error"]; !ok {
			t.Fatalf("expected error detail for closed, got %v", ss.Details)
		}
	})
}
