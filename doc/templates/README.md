# Specification Templates

`doc/templates/slice/` is the canonical starting point for a new VisionOpus
slice. Copy the directory, replace every `{{PLACEHOLDER}}`, and keep the
resulting slice under `doc/slices/<slice-id>/`.

Templates provide structure only. Generated content is not verified evidence,
an approval, or permission to implement.

Run the repository validator after creating or updating a slice:

```bash
./scripts/validate-slices
```

The validator applies structural rules to every slice and readiness gates to
slices marked `ready`, `implementation`, or `complete`.
