# Workflow Analysis

The legacy module directly integrates Warm-Flow 1.8.5 and stores definitions, nodes, skips, tasks, history and user assignments in `flow_*` tables. Controllers expose definition lifecycle, instance start/terminate and approve/reject.

## Migrated Behavior

- Versioned JSONB definitions.
- Instance variables and immutable event history.
- Ordered approval nodes with explicit terminal state.
- Approve, reject, terminate and task ownership checks.
- Optimistic status transitions to prevent double approval.
- Safe condition evaluation restricted to equality, inequality, membership and boolean expressions. No Go/JavaScript evaluation is used.

## Intentional Boundary

The Java engine is not embedded. The initial Go engine is a deterministic persistent state machine; it does not claim Warm-Flow plugin, listener or designer-runtime compatibility. Existing definitions require conversion to the new JSON contract.
