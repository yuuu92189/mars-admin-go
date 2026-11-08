package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mars-admin-go/internal/modules/outbox/contracts"
	sqlc "mars-admin-go/internal/modules/workflow/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/workflow/domain"
	"mars-admin-go/internal/platform/apperror"
	platformoutbox "mars-admin-go/internal/platform/outbox"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateDefinition(ctx context.Context, definition domain.Definition) (domain.Definition, error) {
	document, _ := json.Marshal(definition.Document)
	row, err := sqlc.New(r.pool).CreateWorkflowDefinition(ctx, sqlc.CreateWorkflowDefinitionParams{
		ID: pgUUID(definition.ID), Code: definition.Code, Name: definition.Name,
		Version: definition.Version, Description: definition.Description, Definition: document,
		Status: definition.Status, CreatedBy: optionalUUID(definition.CreatedBy),
		PublishedAt: pgtype.Timestamptz{},
	})
	if err != nil {
		return domain.Definition{}, err
	}
	return mapDefinition(row), nil
}

func (r *Repository) GetDefinition(ctx context.Context, id uuid.UUID) (domain.Definition, error) {
	row, err := sqlc.New(r.pool).GetWorkflowDefinition(ctx, pgUUID(id))
	if err != nil {
		return domain.Definition{}, err
	}
	return mapDefinition(row), nil
}

func (r *Repository) ListDefinitions(ctx context.Context, status string, page, pageSize int) ([]domain.Definition, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListWorkflowDefinitions(ctx, sqlc.ListWorkflowDefinitionsParams{
		Column1: status, Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountWorkflowDefinitions(ctx, status)
	if err != nil {
		return nil, 0, err
	}
	definitions := make([]domain.Definition, 0, len(rows))
	for _, row := range rows {
		definitions = append(definitions, mapDefinition(row))
	}
	return definitions, total, nil
}

func (r *Repository) PublishDefinition(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).PublishWorkflowDefinition(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Start(ctx context.Context, definition domain.Definition, actorID uuid.UUID, input domain.StartInput, node domain.Node) (domain.Instance, domain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Instance{}, domain.Task{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	variables, _ := json.Marshal(input.Variables)
	instanceStatus := "running"
	currentCode := node.Code
	finishedAt := pgtype.Timestamptz{}
	if node.Type == "end" {
		instanceStatus = "approved"
		finishedAt = pgTimestamp(time.Now().UTC())
	}
	txQuery := sqlc.New(tx)
	instanceRow, err := txQuery.CreateWorkflowInstance(ctx, sqlc.CreateWorkflowInstanceParams{
		ID: pgUUID(mustID()), DefinitionID: pgUUID(definition.ID), BusinessID: input.BusinessID,
		InitiatorID: pgUUID(actorID), CurrentNodeCode: pgText(currentCode), Variables: variables,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Instance{}, domain.Task{}, apperror.Conflict("A workflow instance already exists for this businessId.")
		}
		return domain.Instance{}, domain.Task{}, err
	}
	if node.Type == "end" {
		if _, err := tx.Exec(ctx, `UPDATE workflow_instances SET status = $2, finished_at = $3 WHERE id = $1`, instanceRow.ID, instanceStatus, finishedAt); err != nil {
			return domain.Instance{}, domain.Task{}, err
		}
		instanceRow.Status = instanceStatus
		instanceRow.FinishedAt = finishedAt
	}
	task := domain.Task{}
	if node.Type == "approval" {
		taskRow, err := txQuery.CreateWorkflowTask(ctx, sqlc.CreateWorkflowTaskParams{
			ID: pgUUID(mustID()), InstanceID: instanceRow.ID, NodeCode: node.Code, NodeName: node.Name,
			AssigneeID: optionalUUID(node.AssigneeID), AssigneeRole: pgText(node.AssigneeRole),
			DueAt: optionalTimePtr(dueAt(node.DueHours)),
		})
		if err != nil {
			return domain.Instance{}, domain.Task{}, err
		}
		task = mapTask(taskRow)
	}
	if err := txQuery.CreateWorkflowEvent(ctx, sqlc.CreateWorkflowEventParams{
		ID: pgUUID(mustID()), InstanceID: instanceRow.ID, TaskID: pgtype.UUID{},
		ActorUserID: pgUUID(actorID), EventType: "started", FromNodeCode: pgtype.Text{},
		ToNodeCode: pgText(node.Code), Comment: "", Metadata: []byte(`{}`),
	}); err != nil {
		return domain.Instance{}, domain.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Instance{}, domain.Task{}, err
	}
	return mapInstance(instanceRow, definition.Name), task, nil
}

func (r *Repository) GetInstance(ctx context.Context, id uuid.UUID, actor domain.Actor) (domain.Instance, error) {
	var row sqlc.WorkflowInstance
	var name string
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.definition_id, i.business_id, i.initiator_id, i.status,
		       i.current_node_code, i.variables, i.started_at, i.finished_at, d.name
		FROM workflow_instances i
		JOIN workflow_definitions d ON d.id = i.definition_id
		WHERE i.id = $1
		  AND `+visibilitySQL(2, 3, 4), id, actor.UserID, actor.Roles, normalizedScope(actor.DataScope)).Scan(
		&row.ID, &row.DefinitionID, &row.BusinessID, &row.InitiatorID, &row.Status,
		&row.CurrentNodeCode, &row.Variables, &row.StartedAt, &row.FinishedAt, &name,
	)
	if err != nil {
		return domain.Instance{}, err
	}
	return mapInstance(row, name), nil
}

func (r *Repository) ListInstances(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Instance, int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.definition_id, i.business_id, i.initiator_id, i.status,
		       i.current_node_code, i.variables, i.started_at, i.finished_at, d.name,
		       count(*) OVER()
		FROM workflow_instances i
		JOIN workflow_definitions d ON d.id = i.definition_id
		WHERE `+visibilitySQL(1, 2, 3)+`
		ORDER BY i.started_at DESC
		LIMIT $4 OFFSET $5`, actor.UserID, actor.Roles, normalizedScope(actor.DataScope), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	instances := make([]domain.Instance, 0)
	var total int64
	for rows.Next() {
		var row sqlc.WorkflowInstance
		var name string
		if err := rows.Scan(
			&row.ID, &row.DefinitionID, &row.BusinessID, &row.InitiatorID, &row.Status,
			&row.CurrentNodeCode, &row.Variables, &row.StartedAt, &row.FinishedAt, &name, &total,
		); err != nil {
			return nil, 0, err
		}
		instances = append(instances, mapInstance(row, name))
	}
	return instances, total, rows.Err()
}

func (r *Repository) Todo(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListTodoTasks(ctx, sqlc.ListTodoTasksParams{
		AssigneeID: pgUUID(actor.UserID), Column2: actor.Roles,
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountTodoTasks(ctx, sqlc.CountTodoTasksParams{AssigneeID: pgUUID(actor.UserID), Column2: actor.Roles})
	if err != nil {
		return nil, 0, err
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		task := domain.Task{
			ID: uuid.UUID(row.ID.Bytes), InstanceID: uuid.UUID(row.InstanceID.Bytes),
			DefinitionID: uuid.UUID(row.DefinitionID.Bytes), DefinitionName: row.DefinitionName,
			BusinessID: row.BusinessID, NodeCode: row.NodeCode, NodeName: row.NodeName,
			AssigneeRole: text(row.AssigneeRole), Status: row.Status, Decision: text(row.Decision),
			Comment: row.Comment, CreatedAt: row.CreatedAt.Time.UTC(),
		}
		if row.AssigneeID.Valid {
			id := uuid.UUID(row.AssigneeID.Bytes)
			task.AssigneeID = &id
		}
		task.DueAt = timePtr(row.DueAt)
		task.CompletedAt = timePtr(row.CompletedAt)
		tasks = append(tasks, task)
	}
	return tasks, total, nil
}

func (r *Repository) Done(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListDoneTasks(ctx, sqlc.ListDoneTasksParams{
		AssigneeID: pgUUID(actor.UserID), Column2: actor.Roles,
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountDoneTasks(ctx, sqlc.CountDoneTasksParams{AssigneeID: pgUUID(actor.UserID), Column2: actor.Roles})
	if err != nil {
		return nil, 0, err
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		task := domain.Task{
			ID: uuid.UUID(row.ID.Bytes), InstanceID: uuid.UUID(row.InstanceID.Bytes),
			DefinitionID: uuid.UUID(row.DefinitionID.Bytes), DefinitionName: row.DefinitionName,
			BusinessID: row.BusinessID, NodeCode: row.NodeCode, NodeName: row.NodeName,
			AssigneeRole: text(row.AssigneeRole), Status: row.Status, Decision: text(row.Decision),
			Comment: row.Comment, CreatedAt: row.CreatedAt.Time.UTC(),
		}
		if row.AssigneeID.Valid {
			id := uuid.UUID(row.AssigneeID.Bytes)
			task.AssigneeID = &id
		}
		task.DueAt = timePtr(row.DueAt)
		task.CompletedAt = timePtr(row.CompletedAt)
		tasks = append(tasks, task)
	}
	return tasks, total, nil
}

func (r *Repository) TaskContext(ctx context.Context, taskID uuid.UUID, actor domain.Actor) (domain.Task, domain.Instance, domain.Definition, domain.Node, error) {
	var taskRow sqlc.WorkflowTask
	var instanceRow sqlc.WorkflowInstance
	var definitionRow sqlc.WorkflowDefinition
	var definitionName string
	err := r.pool.QueryRow(ctx, `
		SELECT
		    t.id, t.instance_id, t.node_code, t.node_name, t.assignee_id, t.assignee_role,
		    t.status, t.decision, t.comment, t.due_at, t.created_at, t.completed_at,
		    i.id, i.definition_id, i.business_id, i.initiator_id, i.status, i.current_node_code,
		    i.variables, i.started_at, i.finished_at,
		    d.id, d.code, d.name, d.version, d.description, d.definition, d.status, d.created_by,
		    d.created_at, d.updated_at, d.published_at
		FROM workflow_tasks t
		JOIN workflow_instances i ON i.id = t.instance_id
		JOIN workflow_definitions d ON d.id = i.definition_id
		WHERE t.id = $1`, taskID).Scan(
		&taskRow.ID, &taskRow.InstanceID, &taskRow.NodeCode, &taskRow.NodeName, &taskRow.AssigneeID,
		&taskRow.AssigneeRole, &taskRow.Status, &taskRow.Decision, &taskRow.Comment, &taskRow.DueAt,
		&taskRow.CreatedAt, &taskRow.CompletedAt,
		&instanceRow.ID, &instanceRow.DefinitionID, &instanceRow.BusinessID, &instanceRow.InitiatorID,
		&instanceRow.Status, &instanceRow.CurrentNodeCode, &instanceRow.Variables, &instanceRow.StartedAt,
		&instanceRow.FinishedAt,
		&definitionRow.ID, &definitionRow.Code, &definitionRow.Name, &definitionRow.Version,
		&definitionRow.Description, &definitionRow.Definition, &definitionRow.Status, &definitionRow.CreatedBy,
		&definitionRow.CreatedAt, &definitionRow.UpdatedAt, &definitionRow.PublishedAt,
	)
	if err != nil {
		return domain.Task{}, domain.Instance{}, domain.Definition{}, domain.Node{}, err
	}
	definition := mapDefinition(definitionRow)
	definitionName = definition.Name
	var node domain.Node
	for _, candidate := range definition.Document.Nodes {
		if candidate.Code == taskRow.NodeCode {
			node = candidate
			break
		}
	}
	if node.Code == "" || !domain.IsAssignee(node, actor) {
		return domain.Task{}, domain.Instance{}, domain.Definition{}, domain.Node{}, domain.ErrNotAssignee
	}
	return mapTask(taskRow), mapInstance(instanceRow, definitionName), definition, node, nil
}

func (r *Repository) Decide(ctx context.Context, actor domain.Actor, input domain.DecisionInput, decision domain.DOMDecision) (domain.Task, domain.Instance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var taskRow sqlc.WorkflowTask
	var instanceRow sqlc.WorkflowInstance
	var definitionRaw []byte
	var lockedInstanceID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT instance.id
		FROM workflow_instances AS instance
		JOIN workflow_tasks AS task ON task.instance_id = instance.id
		WHERE task.id = $1
		FOR UPDATE OF instance`, input.TaskID).Scan(&lockedInstanceID); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	err = tx.QueryRow(ctx, `
		SELECT
		    t.id, t.instance_id, t.node_code, t.node_name, t.assignee_id, t.assignee_role,
		    t.status, t.decision, t.comment, t.due_at, t.created_at, t.completed_at,
		    i.id, i.definition_id, i.business_id, i.initiator_id, i.status, i.current_node_code,
		    i.variables, i.started_at, i.finished_at, d.definition
		FROM workflow_tasks t
		JOIN workflow_instances i ON i.id = t.instance_id
		JOIN workflow_definitions d ON d.id = i.definition_id
		WHERE t.id = $1
		FOR UPDATE OF t`, input.TaskID).Scan(
		&taskRow.ID, &taskRow.InstanceID, &taskRow.NodeCode, &taskRow.NodeName, &taskRow.AssigneeID,
		&taskRow.AssigneeRole, &taskRow.Status, &taskRow.Decision, &taskRow.Comment, &taskRow.DueAt,
		&taskRow.CreatedAt, &taskRow.CompletedAt,
		&instanceRow.ID, &instanceRow.DefinitionID, &instanceRow.BusinessID, &instanceRow.InitiatorID,
		&instanceRow.Status, &instanceRow.CurrentNodeCode, &instanceRow.Variables, &instanceRow.StartedAt,
		&instanceRow.FinishedAt, &definitionRaw,
	)
	if err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	if taskRow.Status != "pending" {
		return domain.Task{}, domain.Instance{}, domain.ErrDoubleDecision
	}
	if instanceRow.Status != "running" {
		return domain.Task{}, domain.Instance{}, fmt.Errorf("workflow instance is not running")
	}
	var document domain.DefinitionDocument
	if err := json.Unmarshal(definitionRaw, &document); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	node := domain.Node{}
	for _, candidate := range document.Nodes {
		if candidate.Code == taskRow.NodeCode {
			node = candidate
			break
		}
	}
	if !domain.IsAssignee(node, actor) {
		return domain.Task{}, domain.Instance{}, domain.ErrNotAssignee
	}
	variables := map[string]any{}
	_ = json.Unmarshal(instanceRow.Variables, &variables)
	for key, value := range decision.Variables {
		variables[key] = value
	}
	variablesRaw, _ := json.Marshal(variables)
	q := sqlc.New(tx)
	if _, err := q.CompleteWorkflowTask(ctx, sqlc.CompleteWorkflowTaskParams{
		ID: pgUUID(input.TaskID), Decision: pgText(input.Action), Comment: decision.Comment,
	}); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	nextStatus := "running"
	var nextNode *domain.Node
	if decision.TerminalStatus != "" {
		nextStatus = decision.TerminalStatus
	} else {
		nextNode = decision.NextNode
		if nextNode != nil && nextNode.Type == "end" {
			nextStatus = "approved"
			nextNode = nil
		}
	}
	nextCode := ""
	if nextNode != nil {
		nextCode = nextNode.Code
	}
	if _, err := tx.Exec(ctx, `
		UPDATE workflow_instances
		SET status = $2::text, current_node_code = $3::text, variables = $4::jsonb,
		    finished_at = CASE WHEN $2::text = 'running' THEN NULL ELSE now() END
		WHERE id = $1 AND status = 'running'`,
		instanceRow.ID, nextStatus, nullText(nextCode), variablesRaw,
	); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	if nextStatus != "running" {
		if err := q.CancelInstanceTasks(ctx, instanceRow.ID); err != nil {
			return domain.Task{}, domain.Instance{}, err
		}
	}
	if nextNode != nil {
		if _, err := q.CreateWorkflowTask(ctx, sqlc.CreateWorkflowTaskParams{
			ID: pgUUID(mustID()), InstanceID: instanceRow.ID, NodeCode: nextNode.Code, NodeName: nextNode.Name,
			AssigneeID: optionalUUID(nextNode.AssigneeID), AssigneeRole: pgText(nextNode.AssigneeRole),
			DueAt: optionalTimePtr(dueAt(nextNode.DueHours)),
		}); err != nil {
			return domain.Task{}, domain.Instance{}, err
		}
	}
	if err := q.CreateWorkflowEvent(ctx, sqlc.CreateWorkflowEventParams{
		ID: pgUUID(mustID()), InstanceID: instanceRow.ID, TaskID: pgUUID(input.TaskID),
		ActorUserID: pgUUID(actor.UserID), EventType: input.Action, FromNodeCode: pgText(node.Code),
		ToNodeCode: pgText(nextCode), Comment: decision.Comment, Metadata: []byte(`{}`),
	}); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	recipients := []uuid.UUID{uuid.UUID(instanceRow.InitiatorID.Bytes)}
	if nextNode != nil && nextNode.AssigneeID != nil {
		recipients = append(recipients, *nextNode.AssigneeID)
	}
	if nextNode != nil && nextNode.AssigneeRole != "" {
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT role_user.user_id
			FROM access_user_roles AS role_user
			JOIN access_roles AS role ON role.id = role_user.role_id
			WHERE role.code = $1 AND role.status = 'active' AND role.deleted_at IS NULL`,
			nextNode.AssigneeRole)
		if err != nil {
			return domain.Task{}, domain.Instance{}, err
		}
		for rows.Next() {
			var userID uuid.UUID
			if err := rows.Scan(&userID); err != nil {
				rows.Close()
				return domain.Task{}, domain.Instance{}, err
			}
			recipients = append(recipients, userID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return domain.Task{}, domain.Instance{}, err
		}
		rows.Close()
	}
	decisionEvent, err := platformoutbox.NewEvent(
		"workflow_instance", uuid.UUID(instanceRow.ID.Bytes).String(),
		contracts.EventWorkflowDecision, input.TaskID.String(),
		contracts.WorkflowDecision{
			InstanceID: uuid.UUID(instanceRow.ID.Bytes), TaskID: input.TaskID,
			Action: input.Action, NextNode: nextCode, Status: nextStatus,
			Recipients: uniqueUUIDs(recipients), DecidedAt: time.Now().UTC(),
		},
	)
	if err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	if err := platformoutbox.Insert(ctx, tx, decisionEvent); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, domain.Instance{}, err
	}
	taskRow.Status = "completed"
	taskRow.Decision = pgText(input.Action)
	taskRow.Comment = decision.Comment
	taskRow.CompletedAt = pgTimestamp(time.Now().UTC())
	instanceRow.Status = nextStatus
	instanceRow.CurrentNodeCode = nullText(nextCode)
	instanceRow.Variables = variablesRaw
	return mapTask(taskRow), mapInstance(instanceRow, ""), nil
}

func (r *Repository) Terminate(ctx context.Context, instanceID, actorID uuid.UUID, comment string) (domain.Instance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Instance{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM workflow_instances
		WHERE id = $1 AND status = 'running'
		FOR UPDATE`, instanceID).Scan(&lockedID); err != nil {
		return domain.Instance{}, err
	}
	q := sqlc.New(tx)
	row, err := q.GetWorkflowInstance(ctx, pgUUID(instanceID))
	if err != nil {
		return domain.Instance{}, err
	}
	if row.Status != "running" {
		return domain.Instance{}, pgx.ErrNoRows
	}
	result, err := tx.Exec(ctx, `
		UPDATE workflow_instances
		SET status = 'terminated', finished_at = now()
		WHERE id = $1 AND status = 'running'`, instanceID)
	if err != nil {
		return domain.Instance{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.Instance{}, pgx.ErrNoRows
	}
	if err := q.CancelInstanceTasks(ctx, pgUUID(instanceID)); err != nil {
		return domain.Instance{}, err
	}
	if err := q.CreateWorkflowEvent(ctx, sqlc.CreateWorkflowEventParams{
		ID: pgUUID(mustID()), InstanceID: pgUUID(instanceID), TaskID: pgtype.UUID{},
		ActorUserID: pgUUID(actorID), EventType: "terminated", FromNodeCode: row.CurrentNodeCode,
		ToNodeCode: pgtype.Text{}, Comment: comment, Metadata: []byte(`{}`),
	}); err != nil {
		return domain.Instance{}, err
	}
	terminateEvent, err := platformoutbox.NewEvent(
		"workflow_instance", instanceID.String(), contracts.EventWorkflowDecision, "terminated",
		contracts.WorkflowDecision{
			InstanceID: instanceID, Action: "terminate", Status: "terminated",
			Recipients: []uuid.UUID{uuid.UUID(row.InitiatorID.Bytes)}, DecidedAt: time.Now().UTC(),
		},
	)
	if err != nil {
		return domain.Instance{}, err
	}
	if err := platformoutbox.Insert(ctx, tx, terminateEvent); err != nil {
		return domain.Instance{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Instance{}, err
	}
	row.Status = "terminated"
	row.FinishedAt = pgTimestamp(time.Now().UTC())
	return mapInstance(row, ""), nil
}

func (r *Repository) Events(ctx context.Context, instanceID uuid.UUID, actor domain.Actor) ([]domain.Event, error) {
	if _, err := r.GetInstance(ctx, instanceID, actor); err != nil {
		return nil, err
	}
	rows, err := sqlc.New(r.pool).ListWorkflowEvents(ctx, pgUUID(instanceID))
	if err != nil {
		return nil, err
	}
	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		event := domain.Event{
			ID: uuid.UUID(row.ID.Bytes), InstanceID: uuid.UUID(row.InstanceID.Bytes),
			Type: row.EventType, FromNode: text(row.FromNodeCode), ToNode: text(row.ToNodeCode),
			Comment: row.Comment, CreatedAt: row.CreatedAt.Time.UTC(),
		}
		if row.TaskID.Valid {
			id := uuid.UUID(row.TaskID.Bytes)
			event.TaskID = &id
		}
		if row.ActorUserID.Valid {
			id := uuid.UUID(row.ActorUserID.Bytes)
			event.ActorUserID = &id
		}
		_ = json.Unmarshal(row.Metadata, &event.Metadata)
		events = append(events, event)
	}
	return events, nil
}

func visibilitySQL(actorParam, rolesParam, scopeParam int) string {
	return fmt.Sprintf(`(
	$%d::text = 'all'
	OR i.initiator_id = $%d
	OR EXISTS (
		SELECT 1
		FROM workflow_tasks visible_task
		WHERE visible_task.instance_id = i.id
		  AND visible_task.assignee_id = $%d
	)
	OR EXISTS (
		SELECT 1
		FROM workflow_tasks visible_task
		JOIN access_roles visible_role ON visible_role.code = visible_task.assignee_role
		WHERE visible_task.instance_id = i.id
		  AND visible_task.assignee_role = ANY($%d::text[])
		  AND visible_role.status = 'active'
		  AND visible_role.deleted_at IS NULL
	)
	OR (
		$%d::text = 'department'
		AND EXISTS (
			SELECT 1
			FROM organization_memberships mine
			JOIN organization_memberships target ON target.department_id = mine.department_id
			WHERE mine.user_id = $%d AND target.user_id = i.initiator_id
		)
	)
	OR (
		$%d::text = 'department_tree'
		AND EXISTS (
			SELECT 1
			FROM organization_memberships mine
			JOIN organization_departments my_dept ON my_dept.id = mine.department_id
			JOIN organization_memberships target ON target.user_id = i.initiator_id
			JOIN organization_departments target_dept ON target_dept.id = target.department_id
			WHERE mine.user_id = $%d AND target_dept.path <@ my_dept.path
		)
	)
	OR (
		$%d::text = 'custom'
		AND EXISTS (
			SELECT 1
			FROM access_user_roles actor_role
			JOIN access_roles actor_access ON actor_access.id = actor_role.role_id
			JOIN access_role_departments allowed ON allowed.role_id = actor_role.role_id
			JOIN organization_memberships target ON target.department_id = allowed.department_id
			WHERE actor_role.user_id = $%d
			  AND actor_access.status = 'active'
			  AND actor_access.deleted_at IS NULL
			  AND target.user_id = i.initiator_id
		)
	)
)`, scopeParam, actorParam, actorParam, rolesParam, scopeParam, actorParam, scopeParam, actorParam, scopeParam, actorParam)
}

func normalizedScope(scope string) string {
	if scope == "" {
		return "self"
	}
	return scope
}

func mapDefinition(row sqlc.WorkflowDefinition) domain.Definition {
	definition := domain.Definition{
		ID: uuid.UUID(row.ID.Bytes), Code: row.Code, Name: row.Name, Version: row.Version,
		Description: row.Description, Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
		UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Definition, &definition.Document)
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		definition.CreatedBy = &id
	}
	definition.PublishedAt = timePtr(row.PublishedAt)
	return definition
}

func mapInstance(row sqlc.WorkflowInstance, definitionName string) domain.Instance {
	instance := domain.Instance{
		ID: uuid.UUID(row.ID.Bytes), DefinitionID: uuid.UUID(row.DefinitionID.Bytes),
		DefinitionName: definitionName, BusinessID: row.BusinessID, InitiatorID: uuid.UUID(row.InitiatorID.Bytes),
		Status: row.Status, CurrentNodeCode: text(row.CurrentNodeCode), Variables: map[string]any{},
		StartedAt: row.StartedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Variables, &instance.Variables)
	instance.FinishedAt = timePtr(row.FinishedAt)
	return instance
}

func mapTask(row sqlc.WorkflowTask) domain.Task {
	task := domain.Task{
		ID: uuid.UUID(row.ID.Bytes), InstanceID: uuid.UUID(row.InstanceID.Bytes),
		NodeCode: row.NodeCode, NodeName: row.NodeName, AssigneeRole: text(row.AssigneeRole),
		Status: row.Status, Decision: text(row.Decision), Comment: row.Comment,
		CreatedAt: row.CreatedAt.Time.UTC(),
	}
	if row.AssigneeID.Valid {
		id := uuid.UUID(row.AssigneeID.Bytes)
		task.AssigneeID = &id
	}
	task.DueAt = timePtr(row.DueAt)
	task.CompletedAt = timePtr(row.CompletedAt)
	return task
}

func dueAt(hours int32) *time.Time {
	if hours <= 0 {
		return nil
	}
	value := time.Now().UTC().Add(time.Duration(hours) * time.Hour)
	return &value
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func nullText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func text(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func optionalTimePtr(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgTimestamp(*value)
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
