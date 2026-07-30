# Authentication Data Mapping

## Status

Discovery only. PostgreSQL schema decisions have not been approved.

## Candidate Legacy Data

The extraction must confirm definitions, relationships, version tables, and
actual use for at least:

- `user`
- `user_authentication`
- `user_authentication_method`
- `user_pincode`
- Session persistence or revocation tables used by `OESession`
- `institution`
- `site`
- `firm`
- User-to-site assignments
- User-to-firm assignments
- `authitem`
- `authitemchild`
- `authassignment`
- SSO default role and right mappings

## Mapping Record

| Legacy table/column | Meaning | PostgreSQL target | Transformation | Reconciliation | Status |
| --- | --- | --- | --- | --- | --- |
| Not yet verified | N/A | N/A | N/A | N/A | Not started |

## Required Migration Decisions

- Whether legacy numeric IDs are preserved or cross-referenced.
- How password hashes are migrated without weakening security.
- How inactive, locked, stale, local, and SSO users are handled.
- How role inheritance is represented.
- How orphaned assignments are reported.
- How institution, site, and firm assignments are reconciled.
- Which session data is intentionally not migrated.
- How PIN and e-signature dependencies are separated.
