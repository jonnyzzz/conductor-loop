# Data Flow: API & SSE

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/data-flow-api.md`.

## Content Requirements
1. **Request Lifecycle**:
    - Client request -> Middleware (CORS/Log) -> Handler -> Storage/Bus -> Response.
2. **SSE Streaming**:
    - `StreamManager` subscription.
    - Polling loop (100ms) checking Message Bus.
    - Event format (`event: message`, `data: ...`).
3. **Authentication**:
    - Optional API Key check (Bearer/X-API-Key).
    - Exempt paths (`/health`, `/ui/`).
4. **Path Safety**:
    - How `findProjectDir` prevents traversal.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Focus on how the API server handles data and events.
- Name the file `data-flow-api.md`.
