# Authentication And Session State Transitions

## Status

State names below are investigation areas, not accepted legacy facts.

The local-login transitions in this file are provisional extractions from
`AUTH-EXT-001`. They must not be implemented as requirements until the cited
claims receive independent verification.

## Candidate Local Login Flow

| From state | Trigger | Preconditions and checks | To state | Failure result | Evidence |
| --- | --- | --- | --- | --- | --- |
| Unauthenticated | Submit standard login form | Local path is available; required fields and supplied institution/site IDs validate | Credentials pending | Form errors or SSO routing; remains unauthenticated | AUTH-CLAIM-0001, AUTH-CLAIM-0002 |
| Credentials pending | Resolve username in institution/site context | One active exact match, otherwise one active permissive match | Credential selected | Failed audit; remains unauthenticated | AUTH-CLAIM-0003-0005 |
| Credential selected | Authenticate selected local user | Active authentication, `OprnLogin`, valid password, allowed password and lock status | Identity authenticated | Failed audit and possible failed-try update; remains unauthenticated | AUTH-CLAIM-0006-0008 |
| Identity authenticated | Establish web session | At least one firm and a resolvable institution/site/firm context | Authenticated with context | Session setup throws if firm or default-site context cannot be established | AUTH-CLAIM-0009, AUTH-CLAIM-0011 |
| Authenticated with context | Complete successful login | Update last login, audit success, set confirmation/reminder flags, redirect | Authenticated destination | Runtime audit count remains unresolved | AUTH-CLAIM-0002, AUTH-CLAIM-0010 |

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
