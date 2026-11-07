# Migration Risk Register

| Risk | Severity | Mitigation |
|---|---|---|
| Secrets are embedded in the SQL dump | Critical | Treat as exposed, rotate all providers, never import values verbatim, use encrypted settings |
| No Git history or branches | High | Audit only observed files; do not claim dev-master coverage |
| MySQL integer IDs exceed JS safe range | High | Deterministic old-ID to UUIDv7 mapping and external-ID table |
| Data-scope rewrite fails open | High | Typed scope plans with tests that deny on unknown scope |
| Process-local WebSocket state | High | Redis Pub/Sub + PostgreSQL catch-up; process state is only a connection registry |
| Duplicate chat delivery | Medium | `client_message_id` unique index and database sequence cursor |
| Workflow semantics differ | High | Versioned Go definition contract, explicit non-compatible boundary |
| Dump timestamps lack timezone | Medium | Interpret through source timezone and reject ambiguous DST transitions |
| SQL dump includes personal content | High | Restricted import environment, field-level minimization and retention policy |
| Third-party endpoints lack credentials | Medium | Contract tests with mock provider; typed not-configured behavior |
| Legacy menu permissions differ from desired names | Medium | Permission mapping table and compatibility aliases |
| Frontend token handling | High | Preserve temporarily; require cookie migration before public production |
