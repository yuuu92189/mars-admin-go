# Test Strategy

## Automated Checks

- Unit tests cover Argon2id, AES-GCM secret handling, safe workflow conditions, workflow cycle rejection, provider-not-configured behavior and configuration validation.
- `go test -race ./...` checks shared runtime behavior.
- OpenAPI generation and sqlc generation are required build steps.

## Database And Redis Integration

`compose.yaml` starts PostgreSQL and Redis with fixed versions. The migration service applies all Goose migrations and the API readiness endpoint validates both dependencies.

The `integration` build-tag suite runs against real Testcontainers PostgreSQL, Redis and MySQL instances. It covers:

- Apply migrations to an empty PostgreSQL database twice to verify idempotence.
- Create a user and verify the unique username constraint.
- Verify department path movement updates all descendants.
- Verify unauthorized user-list access and self/department data scopes.
- Send a duplicate direct message with the same `clientMessageId` and expect a conflict.
- Publish a notice and verify recipient rows are committed before WebSocket fan-out.
- Queue a scheduler job and verify River persists retries and run history.
- Approve one workflow task twice and verify the second decision conflicts.
- Legacy menu, user-page, unread-count, direct-message and codegen compatibility routes.

## Security Cases

- Missing and expired session credentials.
- Revoked session access.
- Refresh-token replay after rotation.
- CSRF rejection for cookie-authenticated `/api/v1` writes.
- Path traversal and executable upload rejection.
- WebSocket connection without a valid session.
- Cross-user group history access.
- SQL injection strings in search filters.

## Docker Smoke Test

```bash
BOOTSTRAP_ADMIN_PASSWORD='local-smoke-test-password' docker compose up --build -d
curl -fsS http://localhost:8080/health/ready
curl -fsS http://localhost:8080/openapi.yaml >/dev/null
docker compose down
```
