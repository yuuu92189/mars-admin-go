-- +goose Up
CREATE TABLE codegen_imported_tables (
    id UUID PRIMARY KEY,
    schema_name VARCHAR(120) NOT NULL,
    table_name VARCHAR(120) NOT NULL,
    table_comment TEXT NOT NULL DEFAULT '',
    class_name VARCHAR(160) NOT NULL,
    package_name VARCHAR(160) NOT NULL,
    module_name VARCHAR(120) NOT NULL,
    business_name VARCHAR(120) NOT NULL,
    function_name VARCHAR(160) NOT NULL,
    author VARCHAR(120) NOT NULL DEFAULT '',
    gen_type VARCHAR(40) NOT NULL DEFAULT 'zip',
    gen_path TEXT NOT NULL DEFAULT '',
    front_type VARCHAR(40) NOT NULL DEFAULT 'vue3',
    form_layout VARCHAR(40) NOT NULL DEFAULT 'one-column',
    parent_menu_id VARCHAR(120) NOT NULL DEFAULT '',
    remark TEXT NOT NULL DEFAULT '',
    columns JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT codegen_imported_tables_unique UNIQUE (schema_name, table_name)
);

CREATE INDEX codegen_imported_tables_updated_idx ON codegen_imported_tables (updated_at DESC);

-- +goose Down
DROP TABLE IF EXISTS codegen_imported_tables;
