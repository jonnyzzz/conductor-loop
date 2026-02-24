# Task: Prevent run/* Artifact Clutter from Polluting git status

## Context

- **ID**: `task-20260223-155370-run-artifacts-git-hygiene`
- **Priority**: P2
- **Source**: `docs/dev/todos.md`

Run artifacts (under `runs/`, `<project>/<task>/runs/`, `.log` files) frequently appear
in `git status` as untracked files, creating noise during normal development. Example:

```
?? run-claude.log
?? run-codex.log
?? run-gemini.log
?? logs_iter1/
?? logs_iter2/
...
```

This happens because:
1. `.gitignore` does not cover all log/run artifact patterns
2. Developer convenience scripts dump logs to the project root
3. Task directories under `runs/` are not consistently ignored

## Requirements

1. **Update `.gitignore`**: Add patterns to suppress all run artifacts:
   ```
   # run-agent runtime artifacts
   runs/
   *.log
   logs_iter*/
   audit_iter*.log
   facts_iter*.log
   xref_iter*.log
   iter*.txt
   run-*.log
   sleep.log
   ```

2. **Verify**: Run `git status` after updating `.gitignore` — zero untracked run artifacts.

3. **Documentation**: Add a note to `docs/dev/` (or `AGENTS.md`) explaining the artifact
   policy: never commit runtime logs or run directories; they are ephemeral.

4. **No false positives**: Verify that the new patterns do not accidentally ignore important
   project files (test scripts named `*.log`, etc.).

## Acceptance Criteria

- `git status` shows zero `??` entries for `*.log`, `runs/`, `logs_*/` patterns in a
  fresh checkout with active runs.
- `git check-ignore -v runs/` confirms the `runs/` directory is ignored.
- All existing committed files remain tracked (no false negatives).
- No `git add -f` is needed for any legitimate project file.

## Verification

```bash
# After updating .gitignore:
git status --short | grep '??' | grep -E '\.log|runs/|logs_iter'
# Should produce zero output

git check-ignore -v runs/ *.log logs_iter1/
# Should show the .gitignore pattern that covers each

# Verify committed files still tracked
git ls-files | grep -E '\.go|\.ts|\.md' | head -5
```

## Reference Files

- `.gitignore` — root gitignore file
- `AGENTS.md` — contributor conventions
- `docs/dev/todos.md` — feature request origin
