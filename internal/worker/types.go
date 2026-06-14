package worker

import (
	"encoding/json"
	"time"
)

type apiResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Error   any    `json:"error"`
	TraceID string `json:"correlationId"`
}

type HealthCheck struct {
	ID              string `json:"id"`
	ServiceID       string `json:"serviceId"`
	ServiceName     string `json:"serviceName"`
	EnvironmentID   string `json:"environmentId"`
	EnvironmentName string `json:"environmentName"`
	URL             string `json:"url"`
	Method          string `json:"method"`
	IntervalSeconds int    `json:"intervalSeconds"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
}

type EventEnvelope struct {
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	Source        string         `json:"source"`
	OccurredAt    time.Time      `json:"occurredAt"`
	CorrelationID string         `json:"correlationId"`
	Payload       map[string]any `json:"payload"`
}

func (e EventEnvelope) Bytes() []byte {
	b, _ := json.Marshal(e)
	return b
}
