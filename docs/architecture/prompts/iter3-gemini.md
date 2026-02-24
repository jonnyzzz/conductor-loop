# Frontend Architecture

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/frontend-architecture.md`.

## Content Requirements
1. **Dual UI Strategy**:
    - **Primary**: React 18 + TypeScript + JetBrains Ring UI (`frontend/`).
    - **Fallback**: Vanilla JS (`web/src/`).
2. **Build Process**:
    - `npm run build` -> `frontend/dist`.
    - Embedding in Go binary.
3. **Key Features**:
    - SSE Data Subscription (live logs, message bus).
    - Task Tree view.
    - Message Bus view.
4. **API Integration**:
    - `useProjectStats` hook.
    - `ProjectStats.tsx` component.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Describe the frontend structure and build flow.
- Name the file `frontend-architecture.md`.
