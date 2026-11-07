-- name: CreateWorkflowDefinition :one
INSERT INTO workflow_definitions (
    id, code, name, version, description, definition, status, created_by, published_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetWorkflowDefinition :one
SELECT * FROM workflow_definitions WHERE id = $1;

-- name: GetPublishedWorkflowDefinition :one
SELECT * FROM workflow_definitions
WHERE code = $1 AND status = 'published'
ORDER BY version DESC
LIMIT 1;

-- name: ListWorkflowDefinitions :many
SELECT * FROM workflow_definitions
WHERE ($1::text = '' OR status = $1)
ORDER BY code, version DESC
LIMIT $2 OFFSET $3;

-- name: CountWorkflowDefinitions :one
SELECT count(*) FROM workflow_definitions WHERE ($1::text = '' OR status = $1);

-- name: PublishWorkflowDefinition :execrows
UPDATE workflow_definitions
SET status = 'published', published_at = now(), updated_at = now()
WHERE id = $1 AND status = 'draft';

-- name: CreateWorkflowInstance :one
INSERT INTO workflow_instances (
    id, definition_id, business_id, initiator_id, current_node_code, variables
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWorkflowInstance :one
SELECT * FROM workflow_instances WHERE id = $1;

-- name: UpdateWorkflowInstance :execrows
UPDATE workflow_instances
SET status = $2, current_node_code = $3, variables = $4,
    finished_at = CASE WHEN $2 = 'running' THEN NULL ELSE now() END
WHERE id = $1 AND status = 'running';

-- name: CreateWorkflowTask :one
INSERT INTO workflow_tasks (
    id, instance_id, node_code, node_name, assignee_id, assignee_role, due_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetWorkflowTask :one
SELECT * FROM workflow_tasks WHERE id = $1;

-- name: ListTodoTasks :many
SELECT t.*, i.business_id, i.definition_id, d.name AS definition_name
FROM workflow_tasks t
JOIN workflow_instances i ON i.id = t.instance_id
JOIN workflow_definitions d ON d.id = i.definition_id
WHERE t.status = 'pending'
  AND (t.assignee_id = $1 OR t.assignee_role = ANY($2::text[]))
ORDER BY t.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountTodoTasks :one
SELECT count(*)
FROM workflow_tasks t
WHERE t.status = 'pending'
  AND (t.assignee_id = $1 OR t.assignee_role = ANY($2::text[]));

-- name: ListDoneTasks :many
SELECT t.*, i.business_id, i.definition_id, d.name AS definition_name
FROM workflow_tasks t
JOIN workflow_instances i ON i.id = t.instance_id
JOIN workflow_definitions d ON d.id = i.definition_id
WHERE t.status = 'completed'
  AND t.decision IS NOT NULL
  AND (t.assignee_id = $1 OR t.assignee_role = ANY($2::text[]))
ORDER BY t.completed_at DESC
LIMIT $3 OFFSET $4;

-- name: CountDoneTasks :one
SELECT count(*)
FROM workflow_tasks t
WHERE t.status = 'completed'
  AND t.decision IS NOT NULL
  AND (t.assignee_id = $1 OR t.assignee_role = ANY($2::text[]));

-- name: CompleteWorkflowTask :execrows
UPDATE workflow_tasks
SET status = 'completed', decision = $2, comment = $3, completed_at = now()
WHERE id = $1 AND status = 'pending';

-- name: CancelInstanceTasks :exec
UPDATE workflow_tasks SET status = 'cancelled', completed_at = now()
WHERE instance_id = $1 AND status = 'pending';

-- name: CreateWorkflowEvent :exec
INSERT INTO workflow_events (
    id, instance_id, task_id, actor_user_id, event_type,
    from_node_code, to_node_code, comment, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListWorkflowEvents :many
SELECT * FROM workflow_events WHERE instance_id = $1 ORDER BY created_at;
