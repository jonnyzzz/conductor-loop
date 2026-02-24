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

## Build and Embedding Process

The frontend assets are integrated into the Go binary for easy distribution as a single executable.

1.  **Build**: Running npm run build in the frontend/ directory generates optimized static assets in frontend/dist.
2.  **Embedding**: The Go binary uses go:embed to include static assets.
3.  **Serving**: The run-agent serve command (defaulting to port 14355) serves these assets.
    - If frontend/dist/index.html exists, it is served as the primary UI.
    - Otherwise, the server falls back to serving assets from web/src.

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

The frontend communicates with the Go backend via a REST/JSON API based at /api/v1.

### Data Fetching and Hooks
- **useProjectStats**: A custom React hook used to fetch and manage project-level metrics from /api/v1/projects/:project_id/stats.
- **ProjectStats.tsx**: A component that utilizes the useProjectStats hook to display summary cards (e.g., active runs, completion rate).

### Key Endpoints
- GET /api/v1/projects: List all projects.
- GET /api/v1/projects/:project_id/tasks: List tasks for a project.
- GET /api/v1/runs/stream/all: SSE endpoint for the unified run tree visualization.
- GET /api/v1/runs/:run_id/stream: SSE endpoint for individual run logs.
