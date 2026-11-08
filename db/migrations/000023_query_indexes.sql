-- +goose Up
CREATE INDEX scheduler_jobs_due_idx
    ON scheduler_jobs (next_run_at)
    WHERE status = 'active' AND next_run_at IS NOT NULL;

CREATE INDEX storage_files_cleanup_idx
    ON storage_files (deleted_at)
    WHERE status = 'deleted';

CREATE INDEX notification_send_logs_notice_idx
    ON notification_send_logs (notice_id, created_at DESC);

CREATE INDEX system_setting_revisions_setting_idx
    ON system_setting_revisions (setting_id, changed_at DESC);

CREATE INDEX identity_sessions_refresh_active_idx
    ON identity_sessions (refresh_expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX workflow_tasks_role_todo_idx
    ON workflow_tasks (assignee_role, created_at DESC)
    WHERE status = 'pending' AND assignee_role IS NOT NULL;

CREATE INDEX audit_login_events_username_idx
    ON audit_login_events (username, occurred_at DESC);

-- +goose Down
DROP INDEX IF EXISTS audit_login_events_username_idx;
DROP INDEX IF EXISTS workflow_tasks_role_todo_idx;
DROP INDEX IF EXISTS identity_sessions_refresh_active_idx;
DROP INDEX IF EXISTS system_setting_revisions_setting_idx;
DROP INDEX IF EXISTS notification_send_logs_notice_idx;
DROP INDEX IF EXISTS storage_files_cleanup_idx;
DROP INDEX IF EXISTS scheduler_jobs_due_idx;
