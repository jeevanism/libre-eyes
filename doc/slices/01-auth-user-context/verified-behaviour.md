# Verified Behaviour

## Status

Twelve local-login claims have been extracted for `AUTH-EXT-001`. None have
been accepted because independent verification is still pending.

Only independently verified evidence may be added to this document. Existing
feature summaries must be rechecked against the current legacy source.

## Accepted Claims

| Claim ID | Behaviour | Source | Verification |
| --- | --- | --- | --- |
| None | No claims accepted | N/A | Independent review pending |

## Rejected Claims

| Claim ID | Rejected statement | Reason | Verification record |
| --- | --- | --- | --- |
| None | No claims rejected | N/A | N/A |

## Uncertain Claims

| Claim ID | Uncertain statement | Missing evidence | Next action |
| --- | --- | --- | --- |
| AUTH-CLAIM-0001 through AUTH-CLAIM-0011 | Local login validation, identity resolution, credential checks, session context, and audit behaviour | Independent source review | Review each evidence record against the cited current source |
| AUTH-CLAIM-0012 | Current `UserIdentityTest` boolean assertions conflict with the implementation's array return contract | Test execution and version-history reconciliation | Run the legacy test in its supported environment and inspect the contract history |

## Verification Queue

| Claim range | Subject | Current state |
| --- | --- | --- |
| AUTH-CLAIM-0001-0004 | Form validation and credential resolution | Extracted; unverified |
| AUTH-CLAIM-0005-0008 | Authentication rejection, permission, password, and lock checks | Extracted; unverified |
| AUTH-CLAIM-0009-0011 | Session population, successful login audit, and form result handling | Extracted; unverified |
| AUTH-CLAIM-0012 | Legacy test and implementation contract mismatch | Extracted; unverified |
