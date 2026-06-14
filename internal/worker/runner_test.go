package worker

import "testing"

func TestFailureEnvelope(t *testing.T) {
	result := failure(HealthCheck{ID: "check-1", ServiceID: "svc-1", URL: "http://example.invalid"}, 500, "boom")
	if result.Subject != "orion.healthcheck.failed" {
		t.Fatalf("unexpected subject %s", result.Subject)
	}
	if result.Envelope.Payload["healthCheckConfigId"] != "check-1" {
		t.Fatal("missing health check id")
	}
}
