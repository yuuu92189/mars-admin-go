-- +goose Up
CREATE TABLE legacy_id_map (
    legacy_table VARCHAR(120) NOT NULL,
    legacy_id VARCHAR(120) NOT NULL,
    new_uuid UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (legacy_table, legacy_id),
    UNIQUE (legacy_table, new_uuid)
);

CREATE TABLE legacy_import_runs (
    id UUID PRIMARY KEY,
    source_description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    report JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT legacy_import_runs_status_check CHECK (status IN ('running', 'completed', 'failed'))
);

CREATE INDEX legacy_import_runs_started_idx ON legacy_import_runs (started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS legacy_import_runs;
DROP TABLE IF EXISTS legacy_id_map;
