-- +goose Up
CREATE UNIQUE INDEX workflow_instances_definition_business_unique
    ON workflow_instances (definition_id, business_id);

-- +goose Down
DROP INDEX IF EXISTS workflow_instances_definition_business_unique;
