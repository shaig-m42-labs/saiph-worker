# saiph-worker
Background worker service for scheduled jobs, async processing, cleanup tasks, and system automation.

**Language:** ```Go```
**Stack:** `Go, NATS, Redis, HTTP client, cron/scheduler.`

**Responsibilities:**
```
Scheduled health checks
Expired deployment cleanup
Incident reminder jobs
SLO calculation jobs
Old token/session cleanup trigger
Daily reliability summary generation
```

**first feature suggestion:**
```
Every 30 seconds:
- fetch active health check configs from core
- call target health endpoint
- publish healthcheck.passed or healthcheck.failed event
```
