-- name: CreateSchedulerJob :one
INSERT INTO scheduler_jobs (
    id, name, handler, cron_expression, payload, status, max_attempts,
    timeout_seconds, concurrency_key, next_run_at, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSchedulerJob :one
SELECT * FROM scheduler_jobs WHERE id = $1;

-- name: ListSchedulerJobs :many
SELECT * FROM scheduler_jobs
WHERE ($1::text = '' OR status = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSchedulerJobs :one
SELECT count(*) FROM scheduler_jobs WHERE ($1::text = '' OR status = $1);

-- name: UpdateSchedulerJob :one
UPDATE scheduler_jobs
SET name = $2, handler = $3, cron_expression = $4, payload = $5,
    max_attempts = $6, timeout_seconds = $7, concurrency_key = $8,
    next_run_at = $9, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetSchedulerJobStatus :one
UPDATE scheduler_jobs
SET status = $2, next_run_at = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteSchedulerJob :execrows
DELETE FROM scheduler_jobs WHERE id = $1;

-- name: CreateSchedulerRun :one
INSERT INTO scheduler_job_runs (id, job_id, river_job_id, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSchedulerRun :exec
UPDATE scheduler_job_runs
SET status = $2, started_at = $3, finished_at = $4,
    duration_ms = $5, attempt = $6, error_message = $7
WHERE id = $1;

-- name: ListSchedulerRuns :many
SELECT * FROM scheduler_job_runs
WHERE job_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
