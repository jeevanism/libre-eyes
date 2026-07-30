# Permission Matrix

## Status

Draft investigation matrix. Permission names and inheritance semantics must be
verified before this becomes a requirement.

| Operation | Anonymous | Authenticated user | Administrator | Context restrictions | Evidence status |
| --- | --- | --- | --- | --- | --- |
| Submit local login | Candidate | N/A | N/A | Institution may affect method selection | Unverified |
| Logout current session | N/A | Candidate | Candidate | Current session | Unverified |
| Read own user context | No | Candidate | Candidate | Institution/site/firm | Unverified |
| Switch site or firm | No | Candidate | Candidate | User assignments | Unverified |
| Search patients | No | Candidate | Candidate | Institution and clinical permission | Unverified |
| Administer users | No | No by default | Candidate | Institution administration | Unverified |
| Assign administrative role | No | No by default | Candidate | Special restrictions expected | Unverified |
| Generate or view own PIN | No | Candidate | Candidate | Re-authentication expected | Unverified |

## Required Evidence

- Controller access rules.
- `AuthManager` and `AuthItem` evaluation behaviour.
- Seeded `authitem`, `authitemchild`, and `authassignment` data.
- Institution-level role migrations.
- Relevant tests and fixtures.
