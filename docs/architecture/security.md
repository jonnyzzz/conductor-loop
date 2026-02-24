# Security Architecture

This page documents the implemented security model for Conductor Loop runtime paths, based on:

- `internal/api/auth.go`
- `internal/api/path_security.go`
- `internal/api/handlers.go`
- `internal/api/handlers_projects.go`
- `internal/api/handlers_projects_messages.go`
- `internal/webhook/webhook.go`
- `internal/config/config.go`
- `internal/config/api.go`
- `internal/config/tokens.go`
- `internal/config/validation.go`
- `internal/obslog/obslog.go`
- `internal/api/audit.go`
- `README.md`
- `docs/dev/security-review-2026-02-23.md`

## Security Boundaries and Trust Model

- The API server trusts the configured storage root (`rootDir`) and treats all request-provided identifiers (`project_id`, `task_id`, `run_id`, file selectors) as untrusted input.
- Security is filesystem-first: access control and confinement checks are enforced before reading/writing project, task, run, and message bus files.
- Local deployment is the default model. Authentication is optional and opt-in.

## API Authentication Model

Authentication is API-key based and optional.

- Middleware entrypoint: `withAuth` -> `RequireAPIKey`.
- When auth is disabled (`api.auth_enabled` false), middleware passes all requests through.
- When enabled, credentials are accepted via:
  - `Authorization: Bearer <key>`
  - `X-API-Key: <key>`
- Failed auth returns:
  - HTTP `401`
  - `WWW-Authenticate: Bearer realm="conductor"`
  - JSON error body (`valid API key required`)

Exempt paths are always public (even when auth is enabled):

- `/api/v1/health`
- `/api/v1/version`
- `/metrics`
- `/ui/` (prefix match)
- all `OPTIONS` requests (CORS preflight)

Notes:

- `api.auth_enabled` and `api.api_key` are config fields.
- `CONDUCTOR_API_KEY` sets `api.api_key` and forces `api.auth_enabled=true`.
- Default behavior is local no-auth unless a key is explicitly configured/enabled.
- Because `/ui/` is the exempt prefix, use `/ui/` (trailing slash), as documented in `README.md`.

## Path Confinement and Traversal Protections

Path traversal defense is layered.

### 1. Identifier validation

`validateIdentifier`:

- trims input
- URL-unescapes input first
- rejects path separators (`/`, `\`)
- rejects `..`

This blocks encoded traversal payloads such as `%2e%2e` before path joins.

### 2. Root-bounded path construction

- `joinPathWithinRoot(root, segments...)` creates clean paths and enforces root confinement.
- `requirePathWithinRoot(root, target, field)` rejects escaped targets with `403`.
- `pathWithinRoot` uses `filepath.Rel` semantics to confirm target remains under root.

### 3. Handler enforcement

Handlers repeatedly apply these checks before filesystem access:

- Message bus APIs:
  - `/api/v1/messages`
  - `/api/projects/{project}/messages`
  - `/api/projects/{project}/tasks/{task}/messages`
  - stream variants
- File APIs:
  - task file endpoint only allows `TASK.md`
  - run file endpoints only allow `stdout`, `stderr`, `prompt`, `output.md`
  - selected file path is still checked with `requirePathWithinRoot`
- Destructive APIs:
  - run/task/project delete
  - project GC
  - all re-check task/run/project path confinement before `RemoveAll`

### 4. Traversal regression tests

`internal/api/traversal_security_test.go` covers encoded traversal attempts on:

- query IDs (`project_id`, `task_id`)
- project/task message endpoints
- run/task/project delete endpoints

Tests assert blocked requests and preservation of outside-root files.

## Webhook HMAC Signing Model

Webhook notifications (`internal/webhook/webhook.go`) are outbound-only `run_stop` events.

- Delivery is asynchronous and non-blocking.
- Optional event filtering (`webhook.events`).
- Retry model: up to 3 attempts with exponential backoff (`1s`, `2s`).
- Optional integrity/authenticity header when `webhook.secret` is set:
  - `X-Conductor-Signature: sha256=<hex(hmac_sha256(secret, raw_json_body))>`
- No signature header is sent when no secret is configured.

Receiver-side implication:

- Receivers must verify `X-Conductor-Signature` with the same secret.
- There is no built-in nonce/timestamp replay guard in header format; replay protection is receiver responsibility.

## Token and Secret Handling

### Token sources and precedence

Per-agent tokens are loaded from:

- `agents.<name>.token` (inline config)
- `agents.<name>.token_file` (file-based secret)
- `CONDUCTOR_AGENT_<NAME>_TOKEN` environment override

Behavior implemented in `internal/config`:

- Environment token overrides configured token and clears `token_file`.
- `token_file` paths are resolved relative to config directory (with `~` expansion support).
- `token_file` content is trimmed and must be non-empty.
- config validation rejects mixed inline `token` + `token_file` configurations.

Operationally, `token_file`/env are preferred for avoiding committed inline secrets.

### Log and audit redaction controls

Two layers reduce inline secret leakage:

- `internal/obslog` structured log redaction:
  - key-based masking (keys containing token/secret/password/auth/etc.)
  - value pattern masking (Bearer tokens, known token prefixes, JWT-like values, token-like `key=value` patterns)
  - long value truncation
- `internal/api/audit.go` form-submission audit sanitization:
  - sensitive key masking
  - token-pattern masking in free text
  - persisted to `<root>/_audit/form-submissions.jsonl`
  - payloads are sanitized before writing

Audit and logging tests validate that raw secrets are redacted and do not appear in persisted audit output.

## CI and GitHub Actions Security Stance

Runtime code in this page does not enforce CI policy directly. CI/CD posture is documented in `docs/dev/security-review-2026-02-23.md`.

As of remediation documented on 2026-02-23:

- GitHub Actions were pinned to immutable SHAs.
- Workflow permissions were narrowed (write scopes moved to release-publish context).
- Tool/version pinning was tightened.
- Installer integrity checks were added.

This section is documentation-grounded; re-auditing live workflow files is out of scope for this page.

## Threat Model Summary

| Asset / Surface | Threat | Implemented Controls | Residual Risk / Notes |
|---|---|---|---|
| API endpoints | Unauthorized API use | Optional API key middleware; Bearer/X-API-Key checks; 401 responses | Local default is no-auth unless enabled; deployers must enable API key for non-local exposure |
| Health/UI/metrics endpoints | Breaking availability via over-restrictive auth | Explicit auth exemptions for health/version/metrics/UI and OPTIONS | Exempt endpoints remain publicly reachable by design |
| Filesystem under `rootDir` | Path traversal to read/write/delete outside root | `validateIdentifier` + `joinPathWithinRoot` + `requirePathWithinRoot` + repeated per-handler checks | Any new handler must keep using these helpers consistently |
| Message bus files | Cross-project/task bus write/read abuse via crafted IDs | Identifier validation and root-confined bus path resolution before open/read/append | Message content itself is user-controlled; rely on client-side rendering safety for display concerns |
| Task/run file endpoints | Arbitrary file read by path injection | File name allowlists (`TASK.md`, `stdout`, `stderr`, `prompt`, `output.md`) plus root confinement checks | If run metadata file paths are compromised on disk, confinement still blocks outside-root paths |
| Webhook delivery | Payload tampering/spoofing | Optional HMAC-SHA256 signature header over raw JSON body | No built-in replay nonce/timestamp; receiver must enforce replay window/idempotency |
| Logs and audit records | Secret leakage (tokens/keys/passwords) | Pattern+key redaction in `obslog`; sanitized form payload persistence in audit log | Redaction is best-effort pattern matching; avoid logging raw credentials in new code paths |
| Config and repo hygiene | Inline token commits | `token_file` + env override support; docs/security review moved guidance away from inline secrets | Inline `token` still supported for compatibility; process controls remain required |
| CI/CD pipeline | Supply-chain compromise in workflows/actions | Documented remediations in 2026-02-23 review (action pinning, scoped permissions, checksum verification) | This architecture page does not continuously verify workflow drift |
