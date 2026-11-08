# Phase 4 Go Architecture Audit

## Boundary Review

Domain packages do not import HTTP, pgx, Redis or sqlc implementation packages. Application
services own use-case rules and depend on narrow repository ports. PostgreSQL, HTTP, River and
external provider implementations remain adapters. Bootstrap uses explicit constructor
injection.

## Findings And Fixes

- MFA setup could overwrite an enabled secret through an authenticated session. Setup now
  refuses when MFA is already enabled; disable remains the deliberate recovery path.
- Captcha enforcement used a static config flag while public settings used a database setting.
  A `CaptchaPolicy` port now reads the public setting, with the environment flag as a fallback.
- Self-registration ignored the public registration setting. A `RegistrationPolicy` port now
  fails closed when registration is disabled or unavailable.
- Role assignment only rejected super-administrator roles. It now also rejects permissions the
  actor does not hold and data scopes broader than the actor's scope.
- Role mutation can no longer edit a role containing permissions the non-super actor does not
  hold, and role creation/update is bounded by the actor's data scope.
- Menu updates now reject descendant-parent cycles; tree builders also guard existing malformed
  data.
- Workflow termination now uses the same instance visibility predicate as workflow reads.
- Messaging group transitions now reject non-member owner transfer and silent no-op role, mute,
  quit and dissolve operations.
- Client-supplied `system` messages are rejected.

## Residual Architecture Conditions

Scheduler `concurrency_key` is enforced with a PostgreSQL session advisory lock; competing River
jobs are snoozed until the active holder releases the lock.
Payment settlement is not production-certified because the repository has no complete order
ledger/replay-protection model.
