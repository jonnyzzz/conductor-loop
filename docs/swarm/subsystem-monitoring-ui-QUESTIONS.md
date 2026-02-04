# Monitoring & Control UI - Questions

- Q: What is the API contract between the Go backend and the UI (REST/JSON schemas, SSE event formats), and how are TypeScript types generated? 
  Proposed default: REST/JSON + SSE; add OpenAPI or manual type sync later. 
  A: Yes, REST/JSON + SSE. Research for the automated open AI and implement integration tests to make sure typescript can consume the API. It can be Node or Browser based.

- Q: What is the development workflow for the UI (webpack dev server with proxy vs embedded only)? 
  A: webpack-dev-server proxy to Go backend; embed for release.

- Q: Do we need a state management library (Redux/Zustand) or is React Context sufficient for MVP? 
  Proposed default: Context + hooks for MVP. 
  A: Context + hooks for MVP. Keep it simple.

- Q: How should log streaming be exposed in the backend (per-run SSE endpoint vs polling output.md)? 
  A: Just create one SSE endpoint that streams all files line-by-line with nice header messages for each. All new runs should be included. No filtering, we jsut send everything. 

- Q: What path validation rules apply to file-read endpoints (e.g., /api/files) to prevent traversal outside ~/run-agent?
  Proposed default: filepath.Clean + prefix check against storage root; reject symlinks outside root. 
  A: Go backend knows what files to show, there is no way for client to specify a path 
- 
