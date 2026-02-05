# Task: Implement Monitoring UI

**Task ID**: ui-frontend
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: api-rest, api-sse

## Objective
Build React-based web UI for monitoring task execution and viewing logs.

## Specifications
Read: docs/specifications/subsystem-ui-frontend.md

## Required Implementation

### 1. Technology Stack Decision

Research and choose:
- **Framework**: React vs Vue vs Svelte
- **Language**: TypeScript (recommended)
- **Build Tool**: Vite vs Create React App
- **UI Library**: Tailwind CSS vs Material-UI vs Chakra UI
- **SSE Client**: Native EventSource vs library

**Recommendation**: React + TypeScript + Vite + Tailwind CSS (modern, fast, popular)

### 2. Project Structure
Location: `frontend/`
```
frontend/
├── src/
│   ├── components/
│   │   ├── TaskList.tsx
│   │   ├── RunDetail.tsx
│   │   ├── LogViewer.tsx
│   │   ├── MessageBus.tsx
│   │   └── RunTree.tsx
│   ├── hooks/
│   │   ├── useSSE.ts
│   │   ├── useAPI.ts
│   │   └── useWebSocket.ts (future)
│   ├── api/
│   │   └── client.ts
│   ├── types/
│   │   └── index.ts
│   ├── App.tsx
│   └── main.tsx
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

### 3. Core Components

#### TaskList Component
- Display all tasks and runs
- Filter by status (running, completed, failed)
- Sort by start time
- Click to view details

#### RunDetail Component
- Show run metadata (run_id, status, times, exit_code)
- Display run tree (parent-child relationships)
- Link to logs

#### LogViewer Component
- Real-time log streaming via SSE
- Auto-scroll to bottom
- Toggle between stdout/stderr
- Search/filter logs
- ANSI color support (terminal colors)

#### MessageBus Component
- Display all message bus messages
- Filter by agent/run
- Collapsible sections
- Auto-refresh

#### RunTree Component
- Visualize parent-child run relationships
- Expand/collapse nodes
- Color by status
- Click to navigate

### 4. SSE Integration

Custom hook for SSE:
```typescript
function useSSE(url: string, onMessage: (event: MessageEvent) => void) {
  useEffect(() => {
    const eventSource = new EventSource(url);

    eventSource.addEventListener('log', onMessage);
    eventSource.addEventListener('status', onMessage);
    eventSource.addEventListener('message', onMessage);

    eventSource.onerror = (err) => {
      console.error('SSE error:', err);
      // Reconnect logic
    };

    return () => eventSource.close();
  }, [url]);
}
```

### 5. API Client

```typescript
class APIClient {
  private baseURL: string;

  async getTasks(): Promise<Task[]> { ... }
  async getRuns(): Promise<Run[]> { ... }
  async getRunInfo(runId: string): Promise<RunInfo> { ... }
  async getMessages(): Promise<Message[]> { ... }
  async stopRun(runId: string): Promise<void> { ... }
}
```

### 6. Features

**Must Have**:
- Task list view
- Run detail view
- Real-time log streaming
- Message bus viewer
- Status indicators

**Nice to Have**:
- Search/filter logs
- Run tree visualization
- Dark mode toggle
- Export logs
- Run comparison

### 7. Styling
- Responsive design (desktop + mobile)
- Dark theme by default
- Color-coded status (green=success, red=error, yellow=running)
- Monospace font for logs
- Clean, minimal UI

### 8. Tests Required
Location: `frontend/tests/`

**Component Tests** (Vitest + React Testing Library):
- TaskList.test.tsx
- RunDetail.test.tsx
- LogViewer.test.tsx

**E2E Tests** (Playwright):
- test/e2e/ui_test.go (using Playwright MCP)
- Test full user flow: view tasks → click run → see logs

### 9. Development Setup
```bash
cd frontend
npm create vite@latest . -- --template react-ts
npm install
npm install -D tailwindcss postcss autoprefixer
npm install @tanstack/react-query  # for data fetching
npm run dev  # start dev server
```

### 10. Production Build
```bash
npm run build  # outputs to frontend/dist
# Serve via Go HTTP server or nginx
```

## Implementation Steps

1. **Research Phase** (20 minutes)
   - Compare React vs Vue vs Svelte
   - Choose UI library (Tailwind recommended)
   - Document decisions in MESSAGE-BUS.md

2. **Setup Phase** (15 minutes)
   - Create Vite + React + TypeScript project
   - Configure Tailwind CSS
   - Set up project structure

3. **Implementation Phase** (90 minutes)
   - Build TaskList component
   - Build RunDetail component
   - Build LogViewer with SSE
   - Build MessageBus component
   - Wire up API client

4. **Styling Phase** (30 minutes)
   - Apply Tailwind styles
   - Make responsive
   - Add dark theme

5. **Testing Phase** (30 minutes)
   - Write component tests
   - Write E2E test with Playwright
   - Manual browser testing

6. **IntelliJ Checks** (15 minutes)
   - Run linter (ESLint)
   - Check TypeScript errors
   - Verify build

## Success Criteria
- All components rendering
- SSE streaming working in browser
- Logs displaying in real-time
- UI responsive and styled
- All tests passing

## Output
Log to MESSAGE-BUS.md:
- DECISION: Technology stack choices and rationale
- FACT: Frontend implemented
- FACT: SSE streaming working in browser
- FACT: All components tested
