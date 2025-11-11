package health

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hungle45/gokit/conc"
	"github.com/hungle45/gokit/tonic/health/check"
)

type Response struct {
	Status   check.Status                   `json:"status"`
	Services map[string]check.ServiceStatus `json:"services,omitempty"`
	Details  map[string]interface{}         `json:"details,omitempty"`
}

type checkResult struct {
	Name   string
	Status check.ServiceStatus
}

// Default creates a new health check handler without checking any services.
func Default() gin.HandlerFunc {
	return New(nil)
}

// New creates a new health check handler with the provided checkers and options.
func New(checkers []check.Checker, opts ...ConfigOption) gin.HandlerFunc {
	cfg := DefaultConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.JSON(cfg.StatusNotOK, Response{
					Status:   check.StatusDown,
					Services: map[string]check.ServiceStatus{},
					Details: map[string]interface{}{
						"error": fmt.Sprintf("panic recovered: %v", r),
					},
				})
			}
		}()

		group := conc.NewGroup()

		resultChan := make(chan checkResult, len(checkers))
		for _, checker := range checkers {
			checker := checker
			group.Go(func() {
				resultChan <- checkResult{
					Name:   checker.Name(),
					Status: checker.Check(c),
				}
			})
		}

		group.Wait()
		close(resultChan)

		response := Response{
			Status:   check.StatusUp,
			Services: make(map[string]check.ServiceStatus, len(checkers)),
		}
		for result := range resultChan {
			response.Services[result.Name] = result.Status
			if !result.Status.Status.IsUp() {
				response.Status = check.StatusDown
			}
		}

		if response.Status.IsUp() {
			c.JSON(cfg.StatusOK, response)
		} else {
			c.JSON(cfg.StatusNotOK, response)
		}
	}
}

// Ping creates a simple ping handler that responds with "pong".
func Ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	}
}
