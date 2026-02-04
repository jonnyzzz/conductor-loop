# Agent Protocol & Governance - Questions

- Q: When a parent agent delegates to a sub-agent, must the parent block until the child completes, or can it exit and rely on Ralph to manage completion? | Proposed default: Parent blocks to preserve context and return path. | A: TBD.
- Q: When should the system move from root-agent polling to a dedicated message-bus polling service? | Proposed default: Post-MVP, only if message volume or latency requires it. | A: TBD.
