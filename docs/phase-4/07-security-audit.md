# Phase 4 Security Audit

## Authentication

- Argon2id is the default password hash; legacy bcrypt hashes are rehashed on successful login.
- Access and refresh tokens are opaque and stored as hashes.
- Refresh rotation uses CAS, generation checks, grace classification and family revocation.
- Session cookies are HttpOnly and use configured Secure/SameSite values.
- CSRF validation applies to cookie-authenticated unsafe API requests.
- MFA setup cannot replace an enabled secret without a disable flow.
- Captcha and registration policy are enforced server-side from public settings.

## Authorization

- Canonical routes declare permission metadata and server-side permission middleware.
- Non-super administrators cannot assign roles with unheld permissions or broader data scopes.
- Disabled/deleted roles no longer contribute custom data scope or menu visibility.
- Workflow termination and reads use the same visibility predicate.
- File reads are owner-scoped unless the actor is a super administrator.
- Group ownership and membership transitions validate active membership.
- Payment callbacks verify provider signatures, bind provider event IDs, amounts and currencies to
  persisted orders, and reject replayed or conflicting events.

## Sensitive Data

- Sensitive settings use AES-256-GCM when a key is configured.
- Existing sensitive settings remain sensitive when updated, even if the key heuristic changes.
- Logs redact password/token/secret metadata.
- Uploads are bounded, temporary multipart files are removed, and active web content is rejected.

## Explicitly Not Certified

- Real SMS, mail, push, social OAuth, S3/OSS and payment provider credentials.
- Live payment provider credentials and production settlement certification.
- Legacy clients that have not yet adopted Bearer or Cookie authentication; raw headers and
  WebSocket query tokens are rejected by the backend.
- Production cutover and secret rotation of imported legacy data.
