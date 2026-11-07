# Architecture Overview

Mars Admin Go Edition is a modular monolith with separate API, Worker, migration and legacy-import entry points.

## Dependency Direction

```text
HTTP adapter -> application service -> module ports <- PostgreSQL/Redis/provider adapters
```

Domain rules are plain Go. HTTP and PostgreSQL packages may depend on domain types; domain packages do not import Chi, pgx, Redis or SQL. Constructor injection is explicit in `internal/bootstrap`.

## Process Topology

| Process | Responsibility |
|---|---|
| `cmd/api` | HTTP API, OpenAPI, metrics, WebSocket, River job insertion |
| `cmd/worker` | River workers, scheduler scan, PostgreSQL cleanup jobs |
| `cmd/migrate` | Goose migration command |
| `cmd/legacy-import` | controlled legacy dump analysis and future reviewed import |

## Runtime Data Flow

1. Request ID and OpenTelemetry span are created.
2. Authentication resolves an opaque token hash to a durable PostgreSQL session.
3. Route middleware checks the required permission.
4. The application service applies validation and transaction boundaries.
5. PostgreSQL commits business data.
6. Redis receives only ephemeral signals or rate-limit counters.
7. WebSocket fan-out and River work happen after persistence.

## Modules

| Module | Core responsibilities |
|---|---|
| Identity | users, Argon2id credentials, sessions, refresh rotation, captcha, SMS codes, TOTP MFA |
| Access | roles, permissions, menus, super-admin policy, data-scope selection |
| Organization | department tree using recursive path queries, positions, memberships |
| System | dictionaries, encrypted settings, revisions |
| Audit | login and security/operation events |
| Storage | local and S3-compatible object backends, content validation, hashes |
| Messaging | direct/group chat, idempotency, receipts, WebSocket and Redis fan-out |
| Notification | notices, targets, receipts, publish logs |
| Scheduler | managed job definitions, River insertion, execution history |
| Workflow | versioned JSONB definitions, approvals, conditions, events |
| Codegen | PostgreSQL introspection and modern Go/OpenAPI preview |
| Monitoring | Go runtime, database pool and Redis readiness |
| Integrations | SMS/social/payment/push provider contracts and typed not-configured errors |

## Deliberate Exclusions

- Java services and Java frameworks.
- Warm-Flow runtime compatibility.
- SSH web terminal and stored server credentials.
- Elasticsearch, Kafka, NATS and mandatory Kubernetes.
