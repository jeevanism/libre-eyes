# Authentication And Session State Transitions

## Status

State names below are investigation areas, not accepted legacy facts.

## Account State Investigation

Determine and verify:

- Initial account state.
- Current password state.
- Password-expired or stale state.
- Soft-locked state and expiry.
- Hard-locked or administratively disabled state.
- Local versus non-local authentication state.

## Session State Investigation

Determine and verify:

- Unauthenticated.
- Authenticated without complete context.
- Authenticated with institution, site, and firm context.
- Context changed.
- Idle or absolute timeout reached.
- Logged out or revoked.

## Required Transition Record

For every accepted transition, record:

| Field | Required value |
| --- | --- |
| From state | Verified state name |
| Trigger | Route, function, timeout, or administrative action |
| Preconditions | Authentication, permission, and context |
| Database reads | Tables and predicates |
| Database writes | Tables and values |
| Audit event | Event name and attributes |
| To state | Verified resulting state |
| Failure result | Error and retained state |
| Concurrency behaviour | Locking, conflict, or last-write semantics |
| Evidence | Source and accepted verification record |
