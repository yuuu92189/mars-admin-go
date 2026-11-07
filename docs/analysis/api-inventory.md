# Legacy API Inventory

The original admin API is mounted under `/api`. Controllers use method/path mappings rather than a versioned contract. The compatibility adapter in this project preserves the observed frontend aliases.

## Authentication

- `GET /auth/captcha`
- `POST /auth/sms-code`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/info`
- `GET|PUT /auth/profile`
- `POST /auth/password`
- `POST /auth/register`
- `POST /app/auth/login`, `/app/auth/sms-code`, `/app/auth/upload-avatar`
- `GET|PUT /app/auth/profile`, `POST /app/auth/password`
- `POST /web/auth/login`, `/web/auth/logout`
- `GET /web/auth/info`, `/web/auth/login-types`
- `GET /crypto/config`, `/crypto/publicKey`, `POST /crypto/decrypt`

## Administration

- Users: page/detail/create/update/delete/batch delete/reset password/quit/approve/reject/export/import/template.
- Roles: page/detail/list/create/update/delete.
- Menus: tree/list/detail/create/update/delete.
- Departments: tree/list/detail/create/update/delete/move.
- Posts: page/list/detail/create/update/delete/tree/move.
- Dictionaries: type/page/list/detail/create/update/delete; data/page/by-type/detail/create/update/delete.
- Settings: group list/detail/update/refresh/public; test payment/email/SMS; key generation; SMS logs.
- Audit: operation log page/delete/clean; login log page/delete/clean; API access page/statistics.
- Monitoring: online list/force logout; Redis stats/info/keys/delete/value; server info.

## Files

- File and group list/detail/create/update/delete.
- Upload, image upload, page by group, download, preview, text preview, rename, move, batch delete.
- Raw private file access through `/files/**`.

## Messaging

- Direct chat send/history/contacts/users/read/unread/stats/online/clear/block/blacklist.
- Group create/list/detail/update/dissolve/quit/members/add/remove/admin/mute/transfer/message/history.
- Notices page/my/detail/create/update/delete/publish/read/read-all/unread/channels/send logs/retry.
- WebSocket `/ws/message?token=...` with `connected`, `pong`, `chat`, `groupChat`, `notice` and `unread` events.

## Jobs And Workflow

- Job page/detail/create/update/delete/status/run/log page/statistics/clean.
- Workflow definitions page/published/detail/publish/unpublish/delete/activate/deactivate/copy.
- Workflow instances page/start/terminate.
- Workflow tasks todo/done/approve/reject.

## Code Generation And Integrations

- Database table list/import/page/detail/update/delete/preview/download/generate/remove/sync.
- WeChat mini program login, phone retrieval, MP OAuth URL/login, MP menu sync/get/delete.
- Payment, mail, SMS and webhook providers are reached indirectly through settings test endpoints.

The new OpenAPI file is the source of truth for `/api/v1`. Compatibility handlers exist only in the HTTP layer and call the same application services.
