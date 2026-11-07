# ADR 0002: Opaque Durable Sessions

Status: accepted

JWT is not used as the session database. Access and refresh credentials are high-entropy opaque values. PostgreSQL stores only SHA-256 hashes and session metadata. Access tokens are short-lived; refresh tokens rotate transactionally. Browser clients receive HttpOnly cookies, while app clients may receive bearer tokens.

For compatibility, the legacy raw `Authorization` header is accepted. In production, the adapter still creates a durable server-side session.
