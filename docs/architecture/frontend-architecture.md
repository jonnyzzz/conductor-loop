# Frontend Architecture

This document describes the frontend architecture of the Conductor Loop monitoring and control interface.

## Dual UI Strategy

Conductor Loop employs a dual UI strategy to ensure both a rich developer experience and a resilient fallback mechanism.

### Primary UI: React + TypeScript
- **Location**: frontend/
- **Stack**: React 18, TypeScript, JetBrains Ring UI.
- **Styling**: JetBrains Ring UI components with JetBrains Mono font for consistency with the CLI.
- **Build Tool**: Vite.

### Fallback UI: Vanilla JS
- **Location**: web/src/
- **Stack**: Vanilla HTML, CSS, and JavaScript.
- **Purpose**: Provides basic monitoring and control capabilities when the modern React frontend is unavailable or for environments where a lightweight solution is preferred.

## Build and Serving Process

Conductor Loop serves its UI through a combination of embedded assets and filesystem-based serving.

1.  **Build**: Running `npm run build` in the `frontend/` directory generates optimized static assets in `frontend/dist`.
2.  **Embedding**: The Go binary uses `go:embed` to include the baseline fallback UI assets from `web/src`.
3.  **Serving**: The `run-agent serve` command (defaulting to port 14355) serves the UI:
    - **Primary UI**: If `frontend/dist/index.html` exists on the filesystem, it is served as the primary UI (not embedded). This allows for updating the UI without rebuilding the binary.
    - **Fallback UI**: If the primary UI is missing from the filesystem, the server falls back to serving the embedded assets from `web/src`.

## Key Features

### SSE Data Subscription
The UI leverages Server-Sent Events (SSE) for live, real-time updates from the backend:
- **Live Logs**: Progressive streaming of stdout and stderr for active runs.
- **Message Bus**: Real-time updates to project and task-level message buses.
- **Status Updates**: Real-time monitoring of run statuses and task progress.

### Task Tree View
A hierarchical navigation component that organizes work by:
- **Projects**: Root nodes (e.g., conductor-loop).
- **Tasks**: Specific objectives within a project (e.g., task-20260224-095158).
- **Runs**: Individual agent execution attempts, sorted chronologically.

### Message Bus View
A dedicated view for the append-only message bus files:
- **Project Bus**: High-level coordination and fact propagation.
- **Task Bus**: Detailed log of decisions, facts, and progress for a specific task.
- **Interactivity**: Supports posting USER/ANSWER messages back to the backend.

## API Integration

The frontend communicates with the Go backend via a REST/JSON API based at /api.

### Data Fetching and Hooks
- **useProjectStats**: A custom React hook used to fetch and manage project-level metrics from /api/projects/:project_id/stats.
- **ProjectStats.tsx**: A component that utilizes the useProjectStats hook to display summary cards (e.g., active runs, completion rate).

### Key Endpoints
- GET /api/projects: List all projects.
- GET /api/projects/:project_id/tasks: List tasks for a project.
- GET /api/runs/stream/all: SSE endpoint for the unified run tree visualization.
- GET /api/runs/:run_id/stream: SSE endpoint for individual run logs.
