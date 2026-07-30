# Evidence Records

Store one YAML file per bounded extraction claim.

Required format:

```yaml
id: AUTH-CLAIM-0001
task: AUTH-EXT-001
claim: A single, testable statement of legacy behaviour.
source:
  path: OpenEyes/openeyes/protected/path/File.php
  symbol: ClassOrMethod
  lines: 10-25
evidence_type: controller|model|migration|test|fixture|config|runtime
confidence: needs_independent_verification
open_questions: []
```

Optional `supporting_sources` and `contradicting_source` fields may cite
additional evidence without weakening the requirement for one bounded claim.

Rules:

- One behavioural claim per record.
- Cite current source lines, not stale documentation line ranges.
- Include tests or fixtures when they demonstrate the claim.
- Use `unknown` when a symbol or line cannot yet be confirmed.
- Do not put proposed Go behaviour in a legacy evidence record.
- Keep extracted claims at `needs_independent_verification` until a separate
  reviewer records an accepted, rejected, or uncertain result.
