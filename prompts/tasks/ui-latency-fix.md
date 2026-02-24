# Task: Fix Web UI Update Latency

## Context
The current Web UI updates take multiple seconds to reflect changes from the backend. This latency makes it difficult for users to track the progress of agents in real-time. The issue is likely due to a conservative polling interval or inefficient re-rendering of the UI components.

## Goal
Profile the Web UI performance and implement fixes to reduce the update latency.

## Objectives
1.  **Profile**: Identify the specific cause of the delay (network polling, React/component re-render cycles, or backend response time).
2.  **Optimize Polling**: Adjust the polling interval or mechanism to be more responsive (target < 1 second). Consider if Server-Sent Events (SSE) are applicable or if the current polling logic is flawed.
3.  **Optimize Rendering**: Ensure that the task tree and other UI components only re-render when necessary.
4.  **Fix**: Apply the necessary code changes to `internal/ui` or the relevant frontend code.

## Verification
1.  **Baseline**: Measure the time from a backend state change (e.g., a log line written) to the UI update becoming visible.
2.  **Validation**: After applying fixes, repeat the measurement. The time-to-visible-update should be significantly reduced (ideally under 1 second).
3.  **Regression Check**: Ensure that the faster updates do not cause excessive CPU usage on the client or server.
