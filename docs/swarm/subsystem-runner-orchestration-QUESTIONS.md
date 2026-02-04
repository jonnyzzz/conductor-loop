# Runner & Orchestration - Questions

- Q: How are run-agent binary updates/versioning handled in the Go implementation (manual install vs self-update)?
  Proposed default: Manual install/rebuild for MVP. 
  A: Manual install, you start the binary, it is used for all agents, it must not copy the binary, so the update binary will catch up.


- Q: Where is the config.hcl schema/version defined and validated? 
  A: Embed schema in Go binary; validate on startup, add key to extract the schema, create/update the config.hcl to contain necessary comments and defaults. Make sure you return clean and explanatory error messages. Extract that to a dedicated planning doc.
