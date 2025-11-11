package check

import (
	"context"
)

type Checker interface {
	// Name returns the name of the service being checked.
	Name() string
	// Check performs a health check and returns the status an
	Check(ctx context.Context) ServiceStatus
}

type Status string

const (
	StatusUp   Status = "UP"
	StatusDown Status = "DOWN"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsUp() bool {
	return s == StatusUp
}

type ServiceStatus struct {
	Status  Status                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}
