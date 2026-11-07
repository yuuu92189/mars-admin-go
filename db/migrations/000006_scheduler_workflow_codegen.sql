-- +goose Up
CREATE TABLE scheduler_jobs (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    handler VARCHAR(100) NOT NULL,
    cron_expression VARCHAR(120),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'paused',
    max_attempts INTEGER NOT NULL DEFAULT 5,
    timeout_seconds INTEGER NOT NULL DEFAULT 300,
    concurrency_key VARCHAR(120),
    last_enqueued_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scheduler_jobs_status_check CHECK (status IN ('active', 'paused')),
    CONSTRAINT scheduler_jobs_attempts_check CHECK (max_attempts BETWEEN 1 AND 25),
    CONSTRAINT scheduler_jobs_timeout_check CHECK (timeout_seconds BETWEEN 1 AND 86400)
);

CREATE TABLE scheduler_job_runs (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES scheduler_jobs(id) ON DELETE CASCADE,
    river_job_id BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT,
    attempt INTEGER NOT NULL DEFAULT 1,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scheduler_job_runs_status_check CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled'))
);

CREATE INDEX scheduler_job_runs_job_idx ON scheduler_job_runs (job_id, created_at DESC);

CREATE TABLE workflow_definitions (
    id UUID PRIMARY KEY,
    code CITEXT NOT NULL,
    name VARCHAR(150) NOT NULL,
    version INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    definition JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    CONSTRAINT workflow_definitions_code_version_unique UNIQUE (code, version),
    CONSTRAINT workflow_definitions_status_check CHECK (status IN ('draft', 'published', 'inactive'))
);

CREATE INDEX workflow_definitions_definition_gin_idx ON workflow_definitions USING GIN (definition);

CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY,
    definition_id UUID NOT NULL REFERENCES workflow_definitions(id) ON DELETE RESTRICT,
    business_id VARCHAR(120) NOT NULL,
    initiator_id UUID NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    current_node_code VARCHAR(100),
    variables JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    CONSTRAINT workflow_instances_status_check CHECK (status IN ('running', 'approved', 'rejected', 'terminated'))
);

CREATE TABLE workflow_tasks (
    id UUID PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    node_code VARCHAR(100) NOT NULL,
    node_name VARCHAR(150) NOT NULL,
    assignee_id UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    assignee_role VARCHAR(80),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    decision VARCHAR(20),
    comment TEXT NOT NULL DEFAULT '',
    due_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT workflow_tasks_status_check CHECK (status IN ('pending', 'completed', 'cancelled')),
    CONSTRAINT workflow_tasks_decision_check CHECK (decision IS NULL OR decision IN ('approve', 'reject', 'transfer'))
);

CREATE INDEX workflow_tasks_todo_idx
    ON workflow_tasks (assignee_id, created_at DESC)
    WHERE status = 'pending';

CREATE TABLE workflow_events (
    id UUID PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    task_id UUID REFERENCES workflow_tasks(id) ON DELETE SET NULL,
    actor_user_id UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    event_type VARCHAR(40) NOT NULL,
    from_node_code VARCHAR(100),
    to_node_code VARCHAR(100),
    comment TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX workflow_events_instance_idx ON workflow_events (instance_id, created_at);

CREATE TABLE codegen_generations (
    id UUID PRIMARY KEY,
    schema_name VARCHAR(120) NOT NULL,
    table_names TEXT[] NOT NULL,
    options JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'previewed',
    output_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT codegen_generations_status_check CHECK (status IN ('previewed', 'generated', 'failed'))
);

-- +goose Down
DROP TABLE IF EXISTS codegen_generations;
DROP TABLE IF EXISTS workflow_events;
DROP TABLE IF EXISTS workflow_tasks;
DROP TABLE IF EXISTS workflow_instances;
DROP TABLE IF EXISTS workflow_definitions;
DROP TABLE IF EXISTS scheduler_job_runs;
DROP TABLE IF EXISTS scheduler_jobs;
