# Phase 4 Deployment Audit

## Docker

- Multi-stage build with separate API, worker, migration, outbox and legacy-import binaries.
- Production Dockerfile default is distroless `nonroot`.
- Runtime user is UID/GID 65532.
- Healthcheck invokes the API binary with `-healthcheck`.
- Compose uses an explicit Alpine fallback because GCR was unavailable from this host.

## Compose

- PostgreSQL 18.6 and Redis 8.10.1 have healthchecks and named volumes.
- Migration runs before API/worker startup.
- API readiness depends on PostgreSQL and Redis.
- Development passwords are explicitly local defaults and must not be reused in production.
- The Docker build context is restricted to the Go backend inputs; frontend dependencies,
  Next.js build output and test artifacts are excluded from the image context.

The initial Phase 4 rebuild exposed a 600+ MB context caused by unexcluded frontend and test
artifacts. After updating `.dockerignore`, `docker compose build api worker migrate` transferred
1.82 MB of context and completed successfully for all three images.

## Helm

- API and worker deployments include non-root security contexts, read-only root filesystem,
  dropped capabilities, resource requests/limits and probes.
- Existing secrets are referenced through `existingSecret`.
- Ingress is optional and disabled by default.
- `helm lint` and `helm template` pass using the `alpine/helm:3.18.4` container image.

An isolated local restore rehearsal passed against the Compose PostgreSQL 18.6 instance:
the backup was checksummed, restored into a separate database, verified at migration version 25
with representative row counts, and then removed. Evidence is in
[`../migration/cutover-rehearsal-report.json`](../migration/cutover-rehearsal-report.json).

## Not Certified

- No production cluster was connected.
- No live registry pull, backup/restore rehearsal, key rotation or production database cutover was
  performed.
- The distroless runtime image could not be pulled from GCR on this host.
