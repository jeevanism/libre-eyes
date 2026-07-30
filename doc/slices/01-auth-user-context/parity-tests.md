# Authentication Parity Test Plan

## Purpose

Compare supported legacy behaviour with the new implementation while allowing
explicit, reviewed security improvements.

## Required Fixture Categories

- Active local user.
- Invalid username.
- Invalid password.
- Soft-locked user.
- Administratively disabled or hard-locked user.
- Password-stale or expired user.
- User assigned to one institution, site, and firm.
- User assigned to multiple contexts.
- User with direct permission.
- User with inherited permission.
- User without required permission.
- Administrator and non-administrator role assignment cases.
- Local, LDAP, and SSO authentication-method records.
- User with and without a PIN dependency.

All fixtures must be synthetic or properly de-identified.

## Comparison Record

| Test ID | Input fixture | Legacy result | New result | Classification | Status |
| --- | --- | --- | --- | --- | --- |
| None | Fixtures not created | N/A | N/A | N/A | Not started |

## Parity Classifications

- Preserved exactly.
- Preserved with an approved implementation change.
- Intentionally removed.
- Deferred.
- Unknown and blocking.

## Reconciliation

Authentication migration tests must reconcile:

- User counts by status and authentication method.
- Role, task, operation, and assignment counts.
- Orphaned RBAC edges and assignments.
- Institution, site, and firm assignment counts.
- Accounts requiring forced password reset.
- PIN records included, transformed, or intentionally excluded.
