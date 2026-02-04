# Runner & Orchestration - Questions

- Q: How are run-agent binary updates/versioning handled in the Go implementation (manual install vs self-update)? | Proposed default: Manual install/rebuild for MVP. | A: TBD.
- Q: Where is the config.hcl schema/version defined and validated? | Proposed default: Embed schema in Go binary; validate on startup. | A: TBD.
