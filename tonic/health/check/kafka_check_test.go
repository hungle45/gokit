package check

import "testing"

func TestKafkaCheckerName(t *testing.T) {
	c := NewKafkaChecker(nil)
	if c.Name() != "kafka" {
		t.Fatalf("expected kafka name, got %s", c.Name())
	}
}
