# Security Architecture

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/security.md`.

## Content Requirements
1. **Authentication**:
    - Optional API Key (`Authorization: Bearer <key>`).
    - No auth for local use (default).
2. **Path Safety**:
    - Path confinement in `findProjectDir` and `findProjectTaskDir`.
    - Symlink rejection in Message Bus.
3. **Webhook Security**:
    - HMAC-SHA256 signing (`X-Conductor-Signature`).
4. **Token Handling**:
    - Injected as env vars (`ANTHROPIC_API_KEY`, etc.).
    - `token_file` support to avoid inline secrets.
5. **UI Security**:
    - Browser-origin destructive action blocking (403).

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`

## Instructions
- Explain the security model and protections.
- Name the file `security.md`.
