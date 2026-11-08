# Phase 4 Concurrency And Async Audit

## Refresh Sessions

Existing integration coverage verifies:

- concurrent refresh attempts have one successful CAS rotation;
- grace-window retries do not revoke the newly rotated session;
- retries after the grace period revoke the session family;
- new access and refresh tokens remain valid when no replay occurred.

## Outbox And River

- Business writes and outbox insertion share PostgreSQL transactions.
- Relay claims use `FOR UPDATE SKIP LOCKED`, lease tokens and generation checks.
- River insertion and outbox publication share one transaction.
- Lease renewal, stale-lease recovery, duplicate relay workers and consumer retry behavior are
  covered by integration tests.
- Delivery is at-least-once; no exactly-once claim is made for external providers.

## Scheduler And Workflow

- Scheduler tick, run row and River insert are transactional.
- Workflow decisions lock task/instance state and reject duplicate or non-assignee decisions.
- Workflow termination now checks instance visibility before acquiring the termination lock.

## Race Evidence

The Windows host lacks a C compiler. Race Detector was run in the official Linux Go image:

```text
docker run --rm ... golang:1.27.3-bookworm go test -race ./...
```

Result: PASS.

## Scheduler Gate

Scheduler `concurrency_key` is now enforced with a PostgreSQL session advisory lock held for the
entire worker execution. A competing run receives a bounded River snooze instead of overlapping.
The lock is released on completion or connection loss, so process crashes do not leave a permanent
lease. A real PostgreSQL integration test covers same-key exclusion and release.
