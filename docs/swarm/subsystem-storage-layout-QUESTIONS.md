# Storage & Data Layout - Questions

- Q: Is UTF-8 encoding strictly enforced for all text files (TASK.md, TASK_STATE.md, output.md, logs)? | Proposed default: Yes, strict UTF-8 without BOM. | A: TBD.
- Q: Do we need a schema/version field in run-info.yaml (and similar metadata) to support future layout evolution? | Proposed default: Add `version: 1` to run-info.yaml. | A: TBD.
