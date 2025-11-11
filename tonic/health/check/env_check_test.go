package check

import (
	"os"
	"testing"
)

func TestEnvChecker(t *testing.T) {
	cases := []struct {
		name     string
		envName  string
		envValue string
		wantUp   bool
	}{
		{name: "unset", envName: "GOKIT_TEST_ENV_UNSET", envValue: "", wantUp: false},
		{name: "set", envName: "GOKIT_TEST_ENV_SET", envValue: "value", wantUp: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Unsetenv(tc.envName)
			if tc.envValue != "" {
				if err := os.Setenv(tc.envName, tc.envValue); err != nil {
					t.Fatalf("failed to set env: %v", err)
				}
			}

			c := NewEnvChecker(tc.envName)
			ss := c.Check(nil)
			if tc.wantUp && ss.Status != StatusUp {
				t.Fatalf("expected UP, got %v", ss.Status)
			}
			if !tc.wantUp && ss.Status != StatusDown {
				t.Fatalf("expected DOWN, got %v", ss.Status)
			}

			_ = os.Unsetenv(tc.envName)
		})
	}
}

func TestEnvCheckerDetails(t *testing.T) {
	name := "GOKIT_TEST_ENV_DETAILS"
	_ = os.Unsetenv(name)

	c := NewEnvChecker(name)
	ss := c.Check(nil)
	if _, ok := ss.Details["error"]; !ok {
		t.Fatalf("expected error detail when unset, got %v", ss.Details)
	}

	// set and check value
	_ = os.Setenv(name, "v123")
	defer os.Unsetenv(name)
	ss2 := c.Check(nil)
	if val, ok := ss2.Details["value"]; !ok || val != "v123" {
		t.Fatalf("expected value detail to be set, got %v", ss2.Details)
	}
}
