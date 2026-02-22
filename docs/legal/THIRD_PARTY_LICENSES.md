# Third-Party License Inventory

This document summarizes third-party license metadata observed in this
repository as of 2026-02-22.

The project itself is licensed under Apache License 2.0. Third-party
components remain under their original licenses.

## Go modules used by the built binaries

Source: `go-licenses report ./...`

| Module | License |
| --- | --- |
| `github.com/hashicorp/hcl` | `MPL-2.0` |
| `github.com/pkg/errors` | `BSD-2-Clause` |
| `github.com/spf13/cobra` | `Apache-2.0` |
| `github.com/spf13/pflag` | `BSD-3-Clause` |
| `gopkg.in/yaml.v3` | `MIT` |

Assessment:
- No GPL/AGPL/LGPL licenses were detected in the built Go dependency set.
- `github.com/hashicorp/hcl` is `MPL-2.0` and carries file-level copyleft
  obligations for that dependency's covered files.

## Frontend npm dependency summary

Source: `frontend/` via `npx license-checker`.

Production dependency summary (`npx license-checker --summary --production`):
- `MIT`: 56
- `Apache-2.0`: 2
- `Apache*`: 1
- `BSD-2-Clause`: 1
- `BSD-3-Clause`: 1
- `0BSD`: 1
- `MIT*`: 1
- `UNLICENSED`: 1 (the private `frontend` package itself)

Full dependency summary (`npx license-checker --summary`, includes dev deps):
- Additional non-permissive attribution item: `caniuse-lite` under `CC-BY-4.0`.
- Additional permissive variants observed: `BlueOak-1.0.0`, `Python-2.0`,
  `CC0-1.0`.

Assessment:
- No GPL/AGPL/LGPL licenses were detected in frontend dependencies.
- `CC-BY-4.0` appears in the frontend toolchain dependency graph
  (`caniuse-lite`), so attribution should be retained when distributing
  artifacts that include that data.

## Regeneration commands

```bash
# Go
go install github.com/google/go-licenses@v1.6.0
go-licenses report ./...

# Frontend
cd frontend
npx --yes license-checker --summary --production
npx --yes license-checker --summary
```
