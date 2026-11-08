# Monitoring Migration

## Scheduled Jobs

- River-compatible task model.
- Create/edit/delete, pause/resume, run-now.
- Cron builder and payload JSON validation.
- Run history with status, attempt, duration, River job ID, and error message.
- Only handlers accepted by the Go service are offered.

## Operation Logs

- Event filters for action, resource type, and time range.
- Overview metrics, seven-day sample trend, resource swimlane, and detail drawer.
- Metadata is displayed as server-redacted JSON.
- Delete/export are not offered because the canonical API does not support them.

## Login Logs

- Login event pagination and client-side keyword/outcome filtering.
- Success/failure overview and heat grid.
- IP, client type, failure reason, user agent, and detail drawer.
- No delete/export action is shown without backend support.

## Data Scope

Trend and heat views aggregate the latest 100 real events returned by the canonical APIs. The UI labels this sample scope; the final report must not describe it as a full historical aggregate.
