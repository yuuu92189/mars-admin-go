# ADR 0005: Go Workflow State Machine

Status: accepted

The Java Warm-Flow runtime is not embedded. Definitions are versioned JSONB documents validated against a Go-owned schema. Transitions use database transactions and compare-and-set status updates. Conditions use a closed parser rather than dynamic code evaluation.
