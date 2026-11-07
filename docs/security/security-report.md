# Security Report

## Implemented Controls

- Argon2id password hashing with bounded parameters and constant-time comparison.
- High-entropy opaque access/refresh tokens; only SHA-256 hashes persist.
- Durable server-side sessions, refresh rotation and session revocation.
- HttpOnly/SameSite/Secure-configurable cookies and double-submit CSRF for canonical cookie writes.
- Server-side permission middleware on protected APIs.
- PostgreSQL unique, foreign-key and check constraints.
- Parameterized SQL only; sort and filter inputs use closed allowlists.
- Request body limits, HTTP timeouts, CORS allowlist and security headers.
- Login/SMS rate limiting in Redis without treating Redis as business storage.
- File content sniffing, extension allowlist, generated storage keys, SHA-256 and traversal checks.
- Private file access through authenticated IDs; no API path is derived from client filenames.
- Sensitive settings AES-256-GCM encryption and redacted audit metadata.
- Structured logs avoid passwords, bearer tokens and provider secrets.
- TOTP multi-factor authentication with an encrypted stored secret.
- Image and slider captcha challenges with short expiry and one-time consumption.

## Secret Incident

The legacy `mars.sql` contains real-looking SMTP, SMS, cloud storage, payment and social credentials. They must be considered exposed. The new project does not copy them. Rotate every credential at the provider before migration or production deployment.

## Residual Risks

- The compatibility layer accepts header tokens for legacy clients and cannot force the existing UI to migrate immediately.
- Third-party payment/social/SMS credentials were unavailable, so live provider behavior is not certified.
- S3 code is implemented through a MinIO-compatible client but was not exercised against a live bucket.
- Existing legacy SQL data contains personal information and requires a separate approved migration environment.
- Kubernetes/network policy hardening belongs to the deployment platform and is outside this repository's runtime tests.

## Production Requirements

1. Use TLS at the edge and set `AUTH_COOKIE_SECURE=true`.
2. Set `SETTINGS_ENCRYPTION_KEY` from a secret manager.
3. Leave `BOOTSTRAP_ADMIN_PASSWORD` empty after initial provisioning.
4. Replace console SMS; production never enables the console provider automatically.
5. Rotate and re-enter all provider credentials through encrypted settings or environment secrets.
6. Restrict `/metrics`, `/docs` and tracing exporters to trusted operational networks.
