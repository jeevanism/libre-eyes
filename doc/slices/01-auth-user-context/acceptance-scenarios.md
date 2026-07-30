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

### Successful Local Login

```text
Scenario: A permitted local user signs in with a valid password and context
Given: One active local credential resolves for the supplied username and context
And: The local login path is available for the deployment and username
And: The user has OprnLogin and is not blocked by password or soft-lock status
And: An institution, site, and firm context can be selected
When: The standard login form is submitted with a valid password
Then: The web session is authenticated and populated with user context
And: The last successful login timestamp and login audit are written
Legacy evidence: AUTH-CLAIM-0001-0004, AUTH-CLAIM-0006-0010
Modern decision: Pending security and API specification
Parity classification: Candidate parity
```

### Invalid Or Ambiguous Credentials

```text
Scenario: Credential resolution or password validation fails
Given: No active credential, multiple credentials in the chosen match class,
  an invalid password, an inactive credential, or missing OprnLogin
When: The standard login form is submitted
Then: No authenticated web session is created
And: A login-failed audit is written
And: The client receives a non-enumerating error under the modern security policy
Legacy evidence: AUTH-CLAIM-0004-0008, AUTH-CLAIM-0011
Modern decision: Use one generic client-facing failure response
Parity classification: Intentional security change for externally visible errors
```

### Legacy Password Upgrade

```text
Scenario: A valid legacy salted password is presented
Given: The legacy salted password matches the selected credential
When: Local password verification runs
Then: The credential is rehashed with the current legacy PasswordUtils format
And: The legacy salt is cleared and the conversion is audited
Legacy evidence: AUTH-CLAIM-0007
Modern decision: Pending production hash inventory and migration policy
Parity classification: Candidate transitional migration behaviour
```

### Missing Firm Context

```text
Scenario: Valid credentials cannot establish required user context
Given: Password authentication succeeds
And: The user has no available firm
When: Session context is populated
Then: Legacy session setup throws and web-user login does not complete
Legacy evidence: AUTH-CLAIM-0009
Modern decision: Return a controlled failure; exact contract pending AUTH-EXT-004
Parity classification: Intentional reliability change
```

## Remaining Backlog

1. Repeated failure and account soft lock.
2. Lock expiry or administrative unlock.
3. Expired or stale password.
4. Logout and session revocation.
5. Idle and absolute session timeout.
6. Institution selection.
7. Site and firm selection.
8. Context switch and session-scoped data clearing.
9. Permission inherited through the RBAC graph.
10. Permission denied without required operation.
11. Administrator role assignment restriction.
12. Legacy user migration with a supported password hash.
13. Legacy user migration requiring password reset.
14. Deferred LDAP or SSO user presented to the local walking skeleton.
15. PIN action requiring recent re-authentication.

## Security Rule

Legacy behaviour that leaks account existence, stores weak credentials, exposes
PIN material, or permits unsafe session handling must be classified as an
approved security change rather than reproduced for parity.
