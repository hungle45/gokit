package health_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hungle45/gokit/tonic/health"
	"github.com/hungle45/gokit/tonic/health/check"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockChecker struct {
	name   string
	status check.ServiceStatus
}

func (m *mockChecker) Name() string                                  { return m.name }
func (m *mockChecker) Check(ctx context.Context) check.ServiceStatus { return m.status }

func TestPingHandler(t *testing.T) {
	r := gin.New()
	r.GET("/ping", health.Ping())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if msg, ok := body["message"]; !ok || msg != "pong" {
		t.Fatalf("expected message pong, got %v", body)
	}
}

func TestNew_NoCheckers_ReturnsUp(t *testing.T) {
	r := gin.New()
	r.GET("/health", health.New(nil))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp health.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != check.StatusUp {
		t.Fatalf("expected overall status UP, got %v", resp.Status)
	}

	if len(resp.Services) != 0 {
		t.Fatalf("expected no services, got %v", resp.Services)
	}
}

func TestNew_WithCheckers_ReturnsDownWhenAnyFails(t *testing.T) {
	good := &mockChecker{name: "good", status: check.ServiceStatus{Status: check.StatusUp}}
	bad := &mockChecker{name: "bad", status: check.ServiceStatus{Status: check.StatusDown, Details: map[string]interface{}{"err": "boom"}}}

	r := gin.New()
	r.GET("/health", health.New([]check.Checker{good, bad}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Fatalf("expected status 503 when any service is down, got %d", w.Code)
	}

	var resp health.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != check.StatusDown {
		t.Fatalf("expected overall status DOWN, got %v", resp.Status)
	}

	s, ok := resp.Services["bad"]
	if !ok {
		t.Fatalf("expected service 'bad' present in services map: %v", resp.Services)
	}
	if s.Status != check.StatusDown {
		t.Fatalf("expected bad service status DOWN, got %v", s.Status)
	}
}
