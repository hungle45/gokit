package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type redisChecker struct {
	cli          redis.UniversalClient
	includeStats bool
}

func NewRedisChecker(cli redis.UniversalClient, includeStats bool) Checker {
	return &redisChecker{
		cli:          cli,
		includeStats: includeStats,
	}
}

func (r redisChecker) Name() string {
	return "redis"
}

func (r *redisChecker) Check(ctx context.Context) ServiceStatus {
	if err := r.cli.Ping(ctx).Err(); err != nil {
		return ServiceStatus{
			Status: StatusDown,
			Details: map[string]interface{}{
				"error": "Ping failed: " + err.Error(),
			},
		}
	}

	if r.includeStats {
		infoStr, err := r.cli.Info(ctx, "stats").Result()
		if err != nil {
			return ServiceStatus{
				Status: StatusDown,
				Details: map[string]interface{}{
					"error": fmt.Sprintf("Failed to get redis stats: %v", err),
				},
			}
		}

		stats := parseRedisInfo(infoStr)
		return ServiceStatus{
			Status:  StatusUp,
			Details: map[string]interface{}{"stats": stats},
		}
	}

	return ServiceStatus{
		Status: StatusUp,
	}
}

func parseRedisInfo(info string) map[string]string {
	details := make(map[string]string)
	lines := strings.Split(info, "\r\n")

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			details[key] = val
		}
	}
	return details
}
