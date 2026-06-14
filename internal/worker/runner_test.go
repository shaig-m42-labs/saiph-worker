package worker

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/m42-labs/saiph-worker/internal/config"
)

func TestFailureEnvelope(t *testing.T) {
	result := failure(HealthCheck{ID: "check-1", ServiceID: "svc-1", URL: "http://example.invalid"}, 500, "boom")
	if result.Subject != "orion.healthcheck.failed" {
		t.Fatalf("unexpected subject %s", result.Subject)
	}
	if result.Envelope.Payload["healthCheckConfigId"] != "check-1" {
		t.Fatal("missing health check id")
	}
}

func TestExecuteMapsPassedAndFailedChecks(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer okServer.Close()

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	runner := testRunner()
	passed := runner.execute(context.Background(), HealthCheck{ID: "ok", URL: okServer.URL, Method: "GET", TimeoutSeconds: 1})
	if passed.Subject != "orion.healthcheck.passed" {
		t.Fatalf("expected passed subject, got %s", passed.Subject)
	}

	failed := runner.execute(context.Background(), HealthCheck{ID: "fail", URL: failServer.URL, Method: "GET", TimeoutSeconds: 1})
	if failed.Subject != "orion.healthcheck.failed" {
		t.Fatalf("expected failed subject, got %s", failed.Subject)
	}
	if failed.Envelope.Payload["statusCode"] != 500 {
		t.Fatalf("expected status code in payload, got %v", failed.Envelope.Payload["statusCode"])
	}
}

func TestExecuteMapsTimeoutToFailedCheck(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	runner := testRunner()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	result := runner.execute(ctx, HealthCheck{ID: "slow", URL: slowServer.URL, Method: "GET", TimeoutSeconds: 1})
	if result.Subject != "orion.healthcheck.failed" {
		t.Fatalf("expected timeout failure, got %s", result.Subject)
	}
}

func TestFetchChecks(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/health-checks/active" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[{"id":"check-1","serviceId":"svc-1","serviceName":"orders","environmentId":"env-1","environmentName":"prod","url":"http://example.test","method":"GET","intervalSeconds":30,"timeoutSeconds":3}]}`))
	}))
	defer core.Close()

	runner := testRunner()
	runner.cfg.CoreURL = core.URL

	checks, err := runner.fetchChecks(context.Background())
	if err != nil {
		t.Fatalf("fetch checks failed: %v", err)
	}
	if len(checks) != 1 || checks[0].ID != "check-1" {
		t.Fatalf("unexpected checks: %#v", checks)
	}
}

func testRunner() *Runner {
	return &Runner{
		cfg:    config.Config{CheckInterval: time.Second},
		log:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		client: &http.Client{Timeout: time.Second},
	}
}
