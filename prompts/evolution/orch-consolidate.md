# Orchestrator: Consolidate Facts & Docs (Element 1)

## FIRST: Read your operating manuals

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PROMPT_v5_orchestrator.md
```

Also read RLM methodology:
```bash
cat /Users/jonnyzzz/Work/jonnyzzz.com-src/RLM.md
```

## Your Mission

Run **5 iterations** of a 3-agent workgroup to:
1. Ensure all FACTS files are consistent and newer facts win
2. Consolidate ALL project docs under `docs/` folder (nothing important stays in root)
3. Update cross-references after any moves
4. Commit after each iteration

## Iteration Structure (repeat 5 times)

For each iteration, submit 3 parallel sub-agents via `run-agent job` with distinct roles:

### Agent A (codex) — Auditor/Mover
- Read all facts files, check for contradictions, verify newer > older priority
- Move appropriate root-level `.md` files into `docs/` subfolders
- If a source file is already absent from root and already present under `docs/`, treat it as already consolidated; do not recreate it at root.
- Files to consider moving from root:
  - `DEVELOPMENT.md` → `docs/dev/development.md` (verify path + refs if already moved)
  - `ARCHITECTURE-REVIEW-SUMMARY.md` → `docs/dev/architecture-review.md` (verify path + refs if already moved)
  - `DEPENDENCY_ANALYSIS.md` → `docs/dev/dependency-analysis.md` (verify path + refs if already moved)
  - `IMPLEMENTATION-README.md` → `docs/dev/implementation-status.md`
  - `ISSUES.md` → `docs/dev/issues.md`
  - `QUESTIONS.md` → `docs/dev/questions.md`
  - `TODO.md` / `TODOs.md` → `docs/dev/todos.md` (merged)
  - `THE_PROMPT_v5.md` and variants → `docs/workflow/`
  - `THE_PLAN_v5.md` → `docs/workflow/`
  - `output-docs-examples.md` → `docs/dev/output-examples.md`
  - `Instructions.md` → `docs/dev/`
  - Keep in root: `README.md`, `CLAUDE.md`, `AGENTS.md`, `MESSAGE-BUS.md`
- Use `git mv` for moves to preserve history
- Update internal links in all docs that reference moved files
- Commit changes: `git add -A && git commit -m "docs(consolidate): move <files> into docs/"`

### Agent B (gemini) — Facts Reconciler
- Read ALL facts files:
  ```bash
  ls /Users/jonnyzzz/Work/conductor-loop/docs/facts/
  cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-*.md
  ```
- Find contradictions between facts files (e.g., same fact stated differently)
- Newer fact wins: check `[YYYY-MM-DD]` timestamps in each fact entry
- Update FACTS files to remove/correct contradictions
- Update any docs that still contain the stale version of a contradicted fact
- Commit: `git commit -m "docs(facts): reconcile contradictions, newer facts win"`

### Agent C (claude) — Cross-Reference Fixer
- After Agent A moves files, scan ALL remaining docs for broken links
- Fix any `../` relative paths, `[text](path)` links that point to moved files
- Also update CLAUDE.md, AGENTS.md, README.md if they reference files that moved
- Verify: `grep -rn "\.\./\|docs/dev\|docs/workflow" docs/ --include="*.md" | head -30`
- Commit: `git commit -m "docs(xref): fix cross-references after consolidation"`

## Iteration Management

Run iterations sequentially so each builds on the previous:

```bash
cd /Users/jonnyzzz/Work/conductor-loop

for ITER in 1 2 3 4 5; do
  TS=$(date +%Y%m%d-%H%M%S)
  echo "=== Iteration $ITER: $TS ==="

  # Launch 3 agents in parallel
  ./bin/run-agent job --config config.local.yaml --agent codex --project conductor-loop \
    --task "task-${TS}-consolidate-iter${ITER}-audit" \
    --root /Users/jonnyzzz/run-agent --cwd /Users/jonnyzzz/Work/conductor-loop \
    --prompt "Read THE_PROMPT_v5.md and RLM.md first. Then: audit root .md files, move appropriate ones into docs/ subfolders using git mv, update cross-references, commit. Iteration $ITER of 5. Focus on: $([ $ITER -eq 1 ] && echo 'DEVELOPMENT.md ARCHITECTURE-REVIEW-SUMMARY.md DEPENDENCY_ANALYSIS.md' || [ $ITER -eq 2 ] && echo 'ISSUES.md QUESTIONS.md TODO.md TODOs.md' || [ $ITER -eq 3 ] && echo 'THE_PROMPT_v5*.md THE_PLAN_v5.md into docs/workflow/' || [ $ITER -eq 4 ] && echo 'IMPLEMENTATION-README.md Instructions.md output-docs-examples.md' || echo 'final review, fix any remaining root docs, verify all links')" \
    --timeout 20m &
  PID_A=$!

  ./bin/run-agent job --config config.local.yaml --agent gemini --project conductor-loop \
    --task "task-${TS}-consolidate-iter${ITER}-facts" \
    --root /Users/jonnyzzz/run-agent --cwd /Users/jonnyzzz/Work/conductor-loop \
    --prompt "Read THE_PROMPT_v5.md and RLM.md first. Then: read all docs/facts/FACTS-*.md files, find contradictions (newer timestamp wins), fix the FACTS files, update any docs with stale facts. Commit changes. Iteration $ITER of 5." \
    --timeout 20m &
  PID_B=$!

  ./bin/run-agent job --config config.local.yaml --agent claude --project conductor-loop \
    --task "task-${TS}-consolidate-iter${ITER}-xref" \
    --root /Users/jonnyzzz/run-agent --cwd /Users/jonnyzzz/Work/conductor-loop \
    --prompt "Read THE_PROMPT_v5.md and RLM.md first. Then: scan all docs for broken/stale cross-references, fix relative paths, update links to files that have been moved to docs/ subfolders, ensure README.md and CLAUDE.md are up to date. Commit changes. Iteration $ITER of 5." \
    --timeout 20m &
  PID_C=$!

  # Wait for all 3 agents
  wait $PID_A $PID_B $PID_C
  echo "Iteration $ITER complete"
  git -C /Users/jonnyzzz/Work/conductor-loop log --oneline -3
done
```

## Completion

After all 5 iterations:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git log --oneline -10
echo "DONE" > /Users/jonnyzzz/run-agent/conductor-loop/TASK_CONSOLIDATE_DONE
```

Write a final summary to `output.md` covering all moves made and contradictions resolved.
