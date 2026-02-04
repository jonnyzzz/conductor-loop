# Monitoring & Control UI - Questions

- Q: What is the API contract between the Go backend and the UI (REST/JSON schemas, SSE event formats), and how are TypeScript types generated? | Proposed default: REST/JSON + SSE; add OpenAPI or manual type sync later. | A: TBD.
- Q: What is the development workflow for the UI (webpack dev server with proxy vs embedded only)? | Proposed default: webpack-dev-server proxy to Go backend; embed for release. | A: TBD.
- Q: Do we need a state management library (Redux/Zustand) or is React Context sufficient for MVP? | Proposed default: Context + hooks for MVP. | A: TBD.
- Q: How should log streaming be exposed in the backend (per-run SSE endpoint vs polling output.md)? | Proposed default: Per-run SSE when available; polling fallback for MVP. | A: TBD.
- Q: What path validation rules apply to file-read endpoints (e.g., /api/files) to prevent traversal outside ~/run-agent? | Proposed default: filepath.Clean + prefix check against storage root; reject symlinks outside root. | A: TBD.
