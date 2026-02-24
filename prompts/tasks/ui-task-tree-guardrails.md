# Task: Add Task-Tree Regression Guardrails

## Context

The task tree is the main operator surface for understanding run lineage and status. Recent regressions in tree hierarchy and selection behavior have recurred due to limited automated coverage.

Critical logic currently concentrated in:
- `frontend/src/utils/treeBuilder.ts`
- `frontend/src/components/TreePanel.tsx`
- `frontend/src/hooks/useAPI.tsx` (scoped run merging + ancestry continuity)

The frontend test harness is available (`vitest`, `@testing-library/react`) but tree-specific regression tests are missing.

## Scope

Add deterministic regression tests for tree construction and rendering, including threaded parent/previous-run edge cases.

## Requirements

1. Add unit tests for tree-building logic:
- Create `frontend/src/utils/treeBuilder.test.ts`.
- Cover at minimum:
  - root -> task -> run hierarchy construction
  - `parent_run_id` chain preservation
  - `previous_run_id` restart chain behavior
  - blocked/queued/running status mapping
  - selected-task filtering behavior in `selectTreeRuns`

2. Add component-level tests for `TreePanel`:
- Create `frontend/src/components/TreePanel.test.tsx`.
- Verify:
  - expected nodes render for representative payloads
  - collapse/expand state behavior is stable
  - selection highlighting remains correct across updates
  - terminal summary row behavior remains correct

3. Add regression fixtures:
- Introduce fixture data sets for known problematic shapes:
  - cross-task threaded parents
  - deep restart chains
  - tasks with mixed active + terminal runs

4. CI-friendly execution:
- Ensure tests run via existing `npm test` (Vitest) with no browser-only dependencies.
- Keep runtime reasonable for repeated agent sessions.

5. Documentation:
- Add a short section in frontend testing docs (or `frontend/README.md`) listing the new tree regression suite and command.

## Acceptance Criteria

- Tree-builder and TreePanel regression suites exist and pass locally.
- At least one intentionally broken lineage mapping causes test failure (proves guardrail effectiveness).
- Future hierarchy regressions in lineage/selection are caught by automated tests.

## Verification

```bash
cd /Users/jonnyzzz/Work/conductor-loop/frontend

# Run only new tree tests
npm test -- treeBuilder
npm test -- TreePanel

# Full frontend suite + build
npm test
npm run build
```

## Key Files

- `frontend/src/utils/treeBuilder.ts`
- `frontend/src/components/TreePanel.tsx`
- `frontend/src/hooks/useAPI.tsx`
- `frontend/src/test/setup.ts`
