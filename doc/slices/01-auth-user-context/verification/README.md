# Verification Records

Every evidence claim must receive an independent verification record before it
can enter `verified-behaviour.md`.

Required format:

```yaml
id: AUTH-CLAIM-0001
status: accepted|rejected|uncertain
verified_by: reviewer identifier
source_checked:
  path: OpenEyes/openeyes/protected/path/File.php
  symbol: ClassOrMethod
  lines: 10-25
notes: Why the evidence supports, contradicts, or cannot prove the claim.
```

The verifier must re-read the cited source. A summary produced by another agent
is not sufficient verification.
