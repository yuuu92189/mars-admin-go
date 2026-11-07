# Repository Rules

This project is a Go modular monolith. Keep dependencies directed toward application services and domain rules, never from domain code back into HTTP, PostgreSQL, or Redis.

## Required Practices

- Pass `context.Context` across I/O boundaries and honor cancellation.
- Construct dependencies explicitly in `internal/bootstrap`.
- Keep SQL parameterized. Use sqlc-generated functions for stable queries and dynamic SQL only for bounded, whitelisted list filters.
- Store secrets only in environment variables or a secret manager.
- Use `slog` structured fields. Never log passwords, bearer tokens, refresh tokens, cookies, private keys, or unmasked secret settings.
- Encrypt sensitive settings with the configured AES-256-GCM key before persistence.
- Check authorization server-side. Menu visibility is never authorization.
- Keep background goroutines owned by a runnable component with shutdown propagation.
- Add focused tests for every business rule that can reject or mutate persistent state.
