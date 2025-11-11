package check

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type SqlStats struct {
	MaxOpenConnections int   `json:"max_open_connections"`
	OpenConnections    int   `json:"open_connections"`
	InUse              int   `json:"in_use"`
	Idle               int   `json:"idle"`
	WaitCount          int64 `json:"wait_count"`
	WaitDurationNs     int64 `json:"wait_duration_ns"`
	MaxIdleClosed      int64 `json:"max_idle_closed"`
	MaxLifetimeClosed  int64 `json:"max_lifetime_closed"`
}

type sqlChecker struct {
	conn         *sql.DB
	includeStats bool
}

func NewSQLChecker(dbConn *sql.DB, includeStats bool) Checker {
	return &sqlChecker{
		conn:         dbConn,
		includeStats: includeStats,
	}
}

func (s *sqlChecker) Name() string {
	return "sql_database"
}

func (s *sqlChecker) Check(ctx context.Context) ServiceStatus {
	err := s.conn.PingContext(ctx)

	details := map[string]interface{}{
		"driver": s.getDriverName(),
	}

	if err != nil {
		details["error"] = fmt.Sprintf("Ping failed: %v", err)
		return ServiceStatus{
			Status:  StatusDown,
			Details: details,
		}
	}

	if s.includeStats {
		details["stats"] = s.getStats()
	}

	return ServiceStatus{
		Status:  StatusUp,
		Details: details,
	}
}

func (s *sqlChecker) getDriverName() string {
	if s.conn == nil || s.conn.Driver() == nil {
		return "unknown"
	}
	driver := s.conn.Driver()
	return reflect.TypeOf(driver).String()
}

func (s *sqlChecker) getStats() SqlStats {
	stats := s.conn.Stats()
	return SqlStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDurationNs:     stats.WaitDuration.Nanoseconds(),
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}
