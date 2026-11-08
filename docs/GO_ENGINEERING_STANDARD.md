# Mars Admin Go Engineering Standard

This standard applies to every future change in `mars-admin-go`. It is derived from the current modular monolith, PostgreSQL SQL-first, explicit constructor injection and real-dependency testing strategy.

## 1. Go Naming

- Use short package names: `domain`, `application`, `postgres`, `river`, `http`.
- Do not create `utils`, `common`, `helpers`, `base` or `manager` packages for convenience.
- Name interfaces by the capability used by the caller: `SessionReader`, `DeliveryEnqueuer`, `AccessReader`.
- Avoid Java-style `ServiceImpl`, `RepositoryImpl`, `AbstractService` and `BaseController`.
- Prefer `userID`, `groupID`, `createdAt`, `ctx`, `err` and established Go initialisms.

## 2. Package Design

- `domain` contains business types and pure rules. It must not import HTTP, pgx, Redis, River or filesystem adapters.
- `application` owns use-case validation, authorization decisions and transaction-level orchestration.
- `adapter` packages translate external protocols and persistence. They may depend on their module's application/domain packages.
- `platform` contains cross-cutting primitives only: config, database, Redis, HTTP middleware, security, telemetry and Outbox primitives.
- A module must not import another module's generated sqlc package. Use an application-level port or a deliberate read-model package.

## 3. Interface Design

- Define interfaces where they are consumed, not automatically beside every concrete type.
- Keep interfaces small and use-case specific.
- Do not create an interface solely to produce a mock.
- Do not add a dependency injection container. Assemble concrete dependencies in `internal/bootstrap` or the process entry point.
- A large existing interface may be split gradually, without breaking behavior in one large refactor.

## 4. Struct Design

- Keep a struct responsible for one role.
- Do not create `BaseStruct` or inheritance-like embedding.
- Use constructor functions when invariants must be established.
- Do not add Getter/Setter layers around public Go fields.
- Do not store `context.Context` in long-lived structs.

## 5. Error Handling

- Wrap causes with `%w`.
- Use `errors.Is` and `errors.As`; never infer PostgreSQL errors from driver message text when SQLSTATE is available.
- Map business errors through `internal/platform/apperror`.
- Do not ignore an error unless the behavior is explicitly best-effort, and log or document that choice.
- Do not panic for ordinary request, configuration or database failures.
- HTTP adapters must not expose raw SQL, provider or credential errors to clients.

## 6. Context

- Pass `context.Context` as the first parameter on every I/O boundary.
- Use request context for PostgreSQL, Redis, provider calls and logs.
- Use a bounded timeout for background cleanup, provider calls and migration work.
- Use `context.Background()` only at process ownership boundaries or shutdown where no request context exists.
- Honor cancellation in loops and long-running handlers.

## 7. Concurrency

- Every goroutine must have an owner and a termination path.
- Do not start an unbounded goroutine per request without a concurrency limit.
- Protect shared maps with a mutex or use a concurrency-safe abstraction.
- Prefer database atomic operations and CAS over read-then-write race windows.
- Use `FOR UPDATE`, `SKIP LOCKED`, lease ownership or unique constraints where the database is the coordination point.
- Use bounded channels and define behavior when queues are full.
- Test concurrent database workflows with the race detector where practical.

## 8. Database

- PostgreSQL is the source of truth.
- Use parameterized SQL only.
- Prefer sqlc for stable queries; use pgx for dynamic, bounded queries.
- Do not introduce an ORM.
- Add indexes for actual query predicates and ordering.
- Use partial unique indexes when soft-deleted rows may be recreated.
- Keep JSONB for genuinely flexible payloads, not as a replacement for relational columns.
- Check `pgx.ErrNoRows` explicitly.

## 9. Transactions

- A business invariant that spans multiple writes must be committed or rolled back atomically.
- Do not perform slow external HTTP calls inside a PostgreSQL transaction.
- Use the same transaction for business state and Outbox events.
- Use savepoints only for intentional per-row import isolation.
- Return an explicit conflict when optimistic version checks affect zero rows.
- Do not ignore transaction commit errors.

## 10. HTTP Handlers

- Handlers parse input, resolve the actor, call one application method and write the result.
- Business authorization belongs on the server and must not rely on menu visibility or frontend state.
- Validate request body size and JSON shape.
- Keep JSON limits separate from multipart upload limits.
- Do not buffer streaming file responses in compatibility middleware.
- Return `application/problem+json` for canonical errors and preserve legacy envelope only at the legacy boundary.
- Do not log passwords, access tokens, refresh tokens, cookies or private keys.

## 11. Module Boundaries

- `identity`: users, credentials, sessions, MFA, login and refresh.
- `access`: roles, permissions, menus and access policy.
- `organization`: departments, positions and memberships.
- `messaging`: direct/group messages and realtime messaging.
- `notification`: notices, targets, receipts and delivery.
- `storage`: object metadata and object backends.
- `workflow`: definitions, instances, tasks and decisions.
- `scheduler`: scheduled definitions and runs.
- `outbox`: event relay and operational recovery.
- `audit`, `monitoring`, `integrations`, `codegen`: their named responsibilities only.

Cross-module reads must use an explicit read model or public application port. Do not mutate another module's tables directly.

## 12. Testing

- Add focused tests for every rule that rejects, mutates or authorizes persistent state.
- Use real PostgreSQL for transactions, constraints, locking and concurrency.
- Use real Redis for Lua, rate limiting, cache and realtime behavior.
- Use real MySQL only for legacy import compatibility.
- Do not replace a failing real database assertion with a mock that hides the failure.
- Keep integration tests independent of execution order.
- Run `go test -race ./...` for concurrency-sensitive changes.
- Test failure and recovery paths, not only happy paths.

## 13. Logging

- Use `slog` with structured fields.
- Include request ID, trace ID, actor ID and resource ID where available.
- Log errors once at the layer that has actionable context.
- Never log secrets or raw authentication material.
- Use debug logs for high-volume diagnostics and warn/error for actionable failures.

## 14. Dependency Injection

- Construct dependencies explicitly.
- Pass interfaces only where substitution is real and useful.
- Do not introduce service locators or global registries.
- Keep process-specific wiring in `cmd/*` and API wiring in `internal/bootstrap`.
- Validate configuration before opening network connections.

## 15. Security

- Authenticate every protected route.
- Authorize every sensitive operation server-side.
- Treat status, role, user type and ownership as security fields.
- Encrypt sensitive settings and MFA secrets with AES-256-GCM.
- Store encrypted binary values in a text-safe encoding such as base64.
- Require secure cookies and an encryption key in production.
- Verify payment/provider signatures, freshness and replay behavior before claiming settlement.
- Do not claim exactly-once delivery for external providers.

## 16. Migrations

- Never edit an already-applied migration.
- Add the next Goose number.
- Use expand-and-contract for incompatible schema changes.
- Include indexes and constraints in the same migration as the behavior that needs them.
- Test clean install, upgrade, and safe rollback where possible.
- Do not run destructive migration tests against a production database.

## 17. Code Review

Review must check:

- authorization at the server boundary;
- transaction and CAS behavior;
- context cancellation and goroutine ownership;
- request/response size limits;
- secret handling;
- query plans and indexes;
- test evidence for failure and concurrency paths;
- whether an abstraction actually removes complexity.

## 18. Commit Standard

Use focused commits with a clear verb:

```text
fix: enforce group owner protection
fix: stream legacy file downloads
feat: add outbox lease ownership
docs: record engineering audit
```

Before commit:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go test -tags=integration ./... -count=1
golangci-lint run
govulncheck ./...
go tool sqlc generate
docker compose config --quiet
```

If a command cannot run, record the exact environment reason and do not mark it as passed.
