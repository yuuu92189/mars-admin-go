# Frontend Contract Analysis

## Vue Admin

- Axios base URL is `/api`; responses are unwrapped from `{code,message,data}`.
- `Authorization` stores the raw token without a `Bearer` prefix.
- User state persists only the token, then loads roles, permissions and menus from `/auth/info`.
- Dynamic routes use menu `id`, `parentId`, `type=2`, `path` and `component`.
- Buttons use permission strings. API authorization must still be enforced independently.
- Blob downloads bypass the JSON envelope.
- WebSocket connects to `/ws/message?token=...`, sends a ping every 30 seconds and handles `chat`, `groupChat`, `notice` and `unread`.

## UniApp

- API base URL is hard-coded to `http://localhost:8080`.
- Token storage key is `token`; requests send the raw token in `Authorization`.
- Reads `code` and either `message` or `msg`.
- WebSocket uses the same endpoint and reconnects with increasing delay.

## Compatibility Decision

The Go server exposes `/api/v1` as the canonical API and `/api/*` as a narrow compatibility adapter. The adapter serializes the legacy envelope only at the HTTP boundary. New domain and application types do not carry legacy response objects.

The client should migrate to cookie sessions and stop putting tokens in WebSocket query strings. The compatibility path is temporary and documented in `docs/api/frontend-compatibility.md`.
