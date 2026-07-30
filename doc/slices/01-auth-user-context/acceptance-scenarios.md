# Authentication Acceptance Scenarios

## Status

Candidate scenarios only. They must be rewritten from accepted evidence and
approved modern security decisions before implementation.

## Scenario Template

```text
Scenario:
Given:
When:
Then:
And:
Legacy evidence:
Modern decision:
Parity classification:
```

## Candidate Scenario Backlog

1. Successful local login.
2. Invalid username or password.
3. Repeated failure and account soft lock.
4. Lock expiry or administrative unlock.
5. Expired or stale password.
6. Logout and session revocation.
7. Idle and absolute session timeout.
8. Institution selection.
9. Site and firm selection.
10. Context switch and session-scoped data clearing.
11. Permission inherited through the RBAC graph.
12. Permission denied without required operation.
13. Administrator role assignment restriction.
14. Legacy user migration with a supported password hash.
15. Legacy user migration requiring password reset.
16. Deferred LDAP or SSO user presented to the local walking skeleton.
17. PIN action requiring recent re-authentication.

## Security Rule

Legacy behaviour that leaks account existence, stores weak credentials, exposes
PIN material, or permits unsafe session handling must be classified as an
approved security change rather than reproduced for parity.
