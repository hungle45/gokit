package check

import (
	"context"
	"os"
)

type envChecker struct {
	envVarName string
}

func NewEnvChecker(envVarName string) Checker {
	return &envChecker{
		envVarName: envVarName,
	}
}

func (e *envChecker) Name() string {
	return "env_variable"
}

func (e *envChecker) Check(_ context.Context) ServiceStatus {
	envValue := os.Getenv(e.envVarName)
	if envValue == "" {
		return ServiceStatus{
			Status: StatusDown,
			Details: map[string]interface{}{
				"variable": e.envVarName,
				"error":    "environment variable not set",
			},
		}
	}

	return ServiceStatus{
		Status: StatusUp,
		Details: map[string]interface{}{
			"variable": e.envVarName,
			"value":    envValue,
		},
	}
}
