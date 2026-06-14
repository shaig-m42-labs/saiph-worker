# saiph-worker

Scheduled health check worker for Orion Platform V1.

## Behavior

Every `CHECK_INTERVAL_SECONDS` seconds, the worker:

1. Fetches active health checks from `betelgeuse-core`.
2. Calls each target URL.
3. Publishes `orion.healthcheck.passed` or `orion.healthcheck.failed` to NATS.

Core owns incident creation and idempotency.
