# Open Questions

## Blocking Questions

| ID | Question | Owner | Next action | Status |
| --- | --- | --- | --- | --- |
| AUTH-Q-001 | Which identity provider and SSO protocols are required for the first real deployment? | Product/security | Identify deployment target and identity owner | Open |
| AUTH-Q-002 | Must legacy password hashes support transitional login, or will affected users be forced to reset? | Security/product | Inventory production hash types and approve migration policy | Open |
| AUTH-Q-003 | What are the required idle and absolute session timeouts? | Security/governance | Verify legacy configuration and approve modern policy | Open |
| AUTH-Q-004 | Are concurrent sessions allowed, limited, or revoked on a new login? | Security/product | Verify runtime behaviour and deployment policy | Open |
| AUTH-Q-005 | Which institution, site, and firm context must be selected before patient search? | Clinical/product | Verify legacy workflow and validate with users | Open |
| AUTH-Q-006 | Which audit events and attributes are mandatory for authentication and authorization? | Governance/security | Review legacy audit calls and compliance requirements | Open |
| AUTH-Q-007 | Is PIN material part of authentication migration or a later e-signature migration? | Governance/security | Verify cryptographic dependencies and approve boundary | Open |

## Deferred Questions

| ID | Question | Reason deferred | Revisit trigger |
| --- | --- | --- | --- |
| AUTH-Q-101 | What are each customer's LDAP host, certificate, and directory mappings? | Deployment-specific and outside local walking skeleton | First production integration |
| AUTH-Q-102 | Which user-administration screens are required? | Not needed for first read-only walking skeleton | User administration slice |
