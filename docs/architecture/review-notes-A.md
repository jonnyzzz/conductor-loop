# Architecture Review Notes — High-Level Docs (overview.md, components.md, decisions.md)

## Inconsistency Found: API Server ↔ Runner Dependency

**Files affected:** `overview.md` and `components.md`

**overview.md** (Key Design Principles / System Context Diagram section) states:

> "The API server is a read-mostly observer — it never calls back into the runner or agent."

**components.md** (Component Dependencies section) states:

> "API Server (`internal/api/`) depends on: Runner: To trigger 'stop' actions (though it primarily observes state)."

These two statements contradict each other. One says the API server *never* calls into the runner; the other says it depends on the runner to trigger stop actions.

**Recommended fix:** Align both documents. Either:
- Update `overview.md` to acknowledge that the API server does invoke the runner for stop/signal operations, or
- Update `components.md` to clarify that stop actions are delivered via filesystem signals (e.g., writing a signal file) rather than a direct runner function call — if that is the actual implementation.

---

## Minor Notes

- `overview.md` references `[Subsystem Deep-Dives](../dev/subsystems.md)` and `[Developer Architecture](../dev/architecture.md)`. These are outside the `docs/architecture/` directory and not listed in this index — that is expected and intentional, but worth confirming those files exist.
- `decisions.md` documents 6 ADRs. `overview.md` covers the same topics inline (filesystem-first, O_APPEND+flock, CLI agents, YAML, port 14355, PGID). The two documents are consistent in substance; the ADR doc provides the formal rationale.
- The `README.md` index has been restructured into four logical groups: **Overview**, **Data Flow**, **Subsystems**, **Operations** — all 11 architecture files are represented.
