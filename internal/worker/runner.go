package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/m42-labs/saiph-worker/internal/config"
	"github.com/nats-io/nats.go"
)

type Runner struct {
	cfg    config.Config
	log    *slog.Logger
	nats   *nats.Conn
	client *http.Client
}

func New(cfg config.Config, log *slog.Logger) (*Runner, error) {
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Warn("nats_connect_failed", "error", err.Error())
	}
	return &Runner{
		cfg:    cfg,
		log:    log,
		nats:   nc,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (r *Runner) Close() {
	if r.nats != nil {
		r.nats.Close()
	}
}

func (r *Runner) Run(ctx context.Context) {
	r.tick(ctx)
	ticker := time.NewTicker(r.cfg.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	checks, err := r.fetchChecks(ctx)
	if err != nil {
		r.log.Warn("fetch_checks_failed", "error", err.Error())
		return
	}
	for _, check := range checks {
		result := r.execute(ctx, check)
		if err := r.publish(result.Subject, result.Envelope); err != nil {
			r.log.Warn("publish_failed", "subject", result.Subject, "error", err.Error())
		}
	}
}

func (r *Runner) fetchChecks(ctx context.Context) ([]HealthCheck, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(r.cfg.CoreURL, "/")+"/internal/health-checks/active", nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("core returned %d", resp.StatusCode)
	}
	var body apiResponse[[]HealthCheck]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Data, nil
}

type CheckResult struct {
	Subject  string
	Envelope EventEnvelope
}

func (r *Runner) execute(ctx context.Context, check HealthCheck) CheckResult {
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := check.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(checkCtx, method, check.URL, nil)
	if err != nil {
		return failure(check, 0, err.Error())
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return failure(check, 0, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return failure(check, resp.StatusCode, "unhealthy_status")
	}
	return envelope("orion.healthcheck.passed", "healthcheck.passed", check, resp.StatusCode, "")
}

func failure(check HealthCheck, statusCode int, message string) CheckResult {
	return envelope("orion.healthcheck.failed", "healthcheck.failed", check, statusCode, message)
}

func envelope(subject, eventType string, check HealthCheck, statusCode int, err string) CheckResult {
	return CheckResult{
		Subject: subject,
		Envelope: EventEnvelope{
			EventID:       uuid(),
			EventType:     eventType,
			Source:        "saiph-worker",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: uuid(),
			Payload: map[string]any{
				"healthCheckConfigId": check.ID,
				"serviceId":           check.ServiceID,
				"serviceName":         check.ServiceName,
				"environmentId":       check.EnvironmentID,
				"environmentName":     check.EnvironmentName,
				"url":                 check.URL,
				"statusCode":          statusCode,
				"error":               err,
			},
		},
	}
}

func uuid() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}

func (r *Runner) publish(subject string, envelope EventEnvelope) error {
	if r.nats == nil {
		r.log.Info("nats_unavailable_event_logged", "subject", subject, "eventType", envelope.EventType)
		return nil
	}
	if err := r.nats.Publish(subject, envelope.Bytes()); err != nil {
		return err
	}
	return r.nats.FlushTimeout(2 * time.Second)
}
