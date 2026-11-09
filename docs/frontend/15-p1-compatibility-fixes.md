# P1 Frontend Compatibility Fixes

Updated: 2026-11-09

## Registration Policy

The public settings endpoint exposes `site.site.registration_enabled`. Login and registration now
read that key through a shared helper, with legacy aliases retained only as fallback inputs.

Coverage:

- login shows the registration entry when the actual backend key is true
- registration shows the form when the actual backend key is true
- integration checks the public settings response contains the key

## Direct-Chat Realtime Isolation

Incoming `chat` events are appended only when the message belongs to the selected direct
conversation:

- `senderId = selected contact` and `receiverId = current user`
- `senderId = current user` and `receiverId = selected contact`

Group messages and messages for other contacts are ignored. The WebSocket remains connected while
the user switches contacts; filtering uses the latest selected contact without reconnecting.

## WebSocket Proxy

Next.js rewrites `/ws/message` to the configured Go API. Next 16 proxies WebSocket upgrade
requests through this rewrite, so the browser keeps same-origin cookies and Origin behavior.

The Playwright Phase 3 suite now:

- waits for the real `/ws/message` WebSocket
- verifies the connection transitions to `实时连接`
- fails on browser console errors

## Profile Route

`/profile` is the canonical Java-aligned route. `/settings/profile` is a server-side redirect only.
Migration `000026_profile_route.sql` normalizes imported menu paths to `profile`.

## CI

`Frontend E2E` now runs the entire `frontend/e2e` suite, including
`phase3-business-pages.spec.ts`, and backend/internal path changes trigger the workflow.

Playwright remains fully parallel with four workers. Assertions use a 15-second timeout and the
password lifecycle uses a 20-second redirect timeout to tolerate real Argon2 work under concurrent
load.
