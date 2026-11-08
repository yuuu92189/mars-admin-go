# Dashboard Migration

## Implemented

- 12-column Bento layout.
- Greeting, date, login pulse, range selector, peak, average, and period comparison.
- Real login trend from `audit_login_events`.
- Real operation trend and success rate from `audit_events`.
- Real role distribution from `access_roles` and `access_user_roles`.
- Real recent logins and recent audit activity.
- Real online session, user, role, menu, permission, notice, file, message, and outbox counters.
- Refresh action and API error/403 states.
- Recent activity swimlane and visible-menu shortcuts.

## Data Contract

`GET /api/v1/admin/monitoring/dashboard?range=24h|7d|30d` returns:

- Counter fields.
- Login trend and previous-period total.
- Seven-day operation trend.
- Selected-range operation total and success rate.
- Role distribution.
- Recent login and audit activity.

## Integrity Notes

- No random data.
- No vulnerability statistics are shown because the domain does not exist.
- Timezone is explicitly documented as `Asia/Shanghai` in the page while SQL buckets use UTC source timestamps.
- The page displays unavailable/empty states instead of invented chart points.
