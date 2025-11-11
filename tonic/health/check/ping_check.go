package check

import (
	"context"
	"net"
	"time"
)

type tcpPingChecker struct {
	address string
	dialer  *net.Dialer
}

func NewTCPPingChecker(host, port string) Checker {
	return &tcpPingChecker{
		address: net.JoinHostPort(host, port),
		dialer:  &net.Dialer{},
	}
}

func (p *tcpPingChecker) Name() string {
	return "tcp_ping"
}

func (p *tcpPingChecker) Check(ctx context.Context) ServiceStatus {
	start := time.Now()
	conn, err := p.dialer.DialContext(ctx, "tcp", p.address)
	elapsed := time.Since(start)

	if err != nil {
		return ServiceStatus{
			Status: StatusDown,
			Details: map[string]interface{}{
				"address":    p.address,
				"error":      err.Error(),
				"elapsed_ms": elapsed.Milliseconds(),
			},
		}
	}
	defer conn.Close()

	return ServiceStatus{
		Status: StatusUp,
		Details: map[string]interface{}{
			"address":            p.address,
			"time_to_connect_ms": elapsed.Milliseconds(),
		},
	}
}
