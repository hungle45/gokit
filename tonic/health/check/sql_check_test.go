package check

import "testing"

func TestSQLCheckerNameAndDriverUnknown(t *testing.T) {
	s := &sqlChecker{conn: nil}
	if s.Name() != "sql_database" {
		t.Fatalf("expected sql_database, got %s", s.Name())
	}
	if got := s.getDriverName(); got != "unknown" {
		t.Fatalf("expected unknown driver when conn nil, got %s", got)
	}
}
