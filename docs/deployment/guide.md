# Deployment Guide

## Docker Compose

```bash
export BOOTSTRAP_ADMIN_PASSWORD='choose-a-long-password'
export SETTINGS_ENCRYPTION_KEY="$(openssl rand -base64 32)"
docker compose up --build -d
```

The compose stack starts PostgreSQL 18.6, Redis 8.10.1, a migration job, API and Worker. Images are pinned.

The Dockerfile defaults to `gcr.io/distroless/static-debian13:nonroot` as required. In networks where GCR is blocked, the runtime base is parameterized:

```bash
docker build --network=host \
  --build-arg RUNTIME_IMAGE=alpine:3.22 \
  -t mars-admin-go:local .
```

This fallback changes only the runtime base image; the binary remains static and the build remains multi-stage.

## Migrations

```bash
DATABASE_URL='postgres://...' go run ./cmd/migrate up
DATABASE_URL='postgres://...' go run ./cmd/migrate status
DATABASE_URL='postgres://...' go run ./cmd/migrate version
```

Run migrations once before rolling API instances. River schema migrations run automatically when the Worker starts.

## Backups

Use continuous PostgreSQL WAL archiving plus daily logical backups:

```bash
DATABASE_URL='postgres://...' scripts/backup.sh backup/mars-admin.dump
ALLOW_DESTRUCTIVE_RESTORE=1 DATABASE_URL='postgres://...' scripts/restore.sh backup/mars-admin.dump
```

The backup script writes a SHA-256 sidecar and restore verifies it before touching the target.
Never run restore against the source database. Use
[`../migration/cutover-rehearsal.md`](../migration/cutover-rehearsal.md) to validate an isolated
target first.

Back up encrypted secret-manager keys separately from database dumps. Redis persistence is useful
for rate limits and ephemeral realtime recovery, but is not a substitute for PostgreSQL backups.
For WeChat Pay callbacks, provide `WECHAT_PAY_API_V3_KEY`; without it callback decryption is
explicitly not configured rather than silently accepting unverified payloads.

## Observability

- Scrape `/metrics`.
- Send API and Worker JSON logs to the platform log pipeline.
- Terminate TLS before the API and preserve `X-Request-ID` and W3C `traceparent` headers.
- Set `OTEL_EXPORTER_OTLP_ENDPOINT` for trace export.
- Alert on readiness failures, HTTP 5xx rate, River discarded jobs and database pool saturation.

## Helm

The optional chart is in `deploy/helm/mars-admin-go`. It expects a pre-created database/Redis URL secret and does not install PostgreSQL or Redis implicitly.

```bash
helm upgrade --install mars-admin deploy/helm/mars-admin-go \
  --set image.repository=registry.example.com/mars-admin-go \
  --set image.tag=1.0.0 \
  --set existingSecret=mars-admin-secrets
```
