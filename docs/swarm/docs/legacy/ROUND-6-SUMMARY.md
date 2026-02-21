# Round 6 Planning Summary

**Date**: 2026-02-04
**Agent**: Claude Sonnet 4.5
**Focus**: Consolidate answered questions, create schema specifications, conduct technical research

## Work Completed

### 1. Consolidated Answered Questions

Processed all `*-QUESTIONS.md` files and integrated answered questions into their respective subsystem specifications:

#### Resolved and Integrated:

**subsystem-storage-layout.md**:
- ✅ UTF-8 encoding (strict, without BOM) for all text files
- ✅ Schema versioning: added `version: 1` field to run-info.yaml
- ✅ Created dedicated schema specification: `subsystem-storage-layout-run-info-schema.md`

**subsystem-env-contract.md**:
- ✅ Path normalization: OS-native using Go filepath.Clean
- ✅ Environment inheritance: full inheritance in MVP (no sandbox)
- ✅ Signal handling: SIGTERM → 30s grace period → SIGKILL
- ✅ Date/time injection: NOT injected (agents access system time themselves)

**subsystem-message-bus-tools.md**:
- ✅ msg_id return value: prints to stdout on success for chaining

**subsystem-message-bus-object-model.md**:
- ✅ Cross-scope parents: allowed, UI resolves across buses with scope labels

**subsystem-monitoring-ui.md**:
- ✅ API contract: REST/JSON + SSE with integration tests for TypeScript consumption
- ✅ Dev workflow: webpack-dev-server proxy to Go backend; embed for release
- ✅ State management: React Context + hooks (no Redux/Zustand in MVP)
- ✅ Log streaming: single SSE endpoint streams all files line-by-line
- ✅ Path validation: Go backend controls allowed paths (no client-specified paths)

**subsystem-runner-orchestration.md**:
- ✅ Binary updates: manual install/rebuild for MVP
- ✅ Config validation: embedded schema in Go binary, `run-agent config` commands
- ✅ Created dedicated schema specification: `subsystem-runner-orchestration-config-schema.md`

**subsystem-agent-protocol.md**:
- ✅ Parent blocking behavior: agent-dependent (can exit, parallel, or wait)
- ✅ Message bus polling: root agent responsibility in MVP

**subsystem-agent-backend-perplexity.md**:
- ✅ Output convention: uses stdout (agent-stdout.txt), no output.md for Perplexity
- ✅ Citations: included inline in response text if API supports

**subsystem-agent-backend-xai.md**:
- ✅ Parked to backlog in ISSUES.md (post-MVP)

#### Remaining Open Questions:

- **subsystem-agent-backend-claude.md**: Needs experimentation to determine correct CLI flags
- **subsystem-agent-backend-codex.md**: Needs experimentation for OPENAI_API_KEY mapping and flags
- **subsystem-agent-backend-gemini.md**: GEMINI_API_KEY confirmed; needs streaming behavior experimentation
- **subsystem-agent-backend-perplexity.md**: Needs research on streaming API capabilities

### 2. Created New Specification Documents

#### subsystem-storage-layout-run-info-schema.md
Complete schema definition for run-info.yaml version 1:
- All required fields (21 total): identity, lineage, agent, timing, paths
- Optional fields: backend metadata, command line
- Field constraints and validation rules
- Go struct definitions with yaml tags
- Schema evolution strategy
- Example complete file

#### subsystem-runner-orchestration-config-schema.md
Complete schema definition for config.hcl:
- Top-level blocks: global settings, ralph, agent_selection, monitoring, delegation
- Agent block schema for CLI and REST backends
- Token management (@file references)
- Validation rules and error message guidelines
- Default configuration template
- Go struct definitions with hcl tags
- `run-agent config` commands: schema, init, validate

#### RESEARCH-FINDINGS.md
Consolidated technical research from WebSearch (Perplexity MCP still 401):

**Go HCL Libraries**:
- Use `github.com/hashicorp/hcl/v2`
- `hclsimple` for direct struct loading
- `gohcl` for struct tag schema
- `hcl-lang/validator` for validation
- Best practices for readable config

**JetBrains Ring UI**:
- NPM installation: `@jetbrains/ring-ui --save-exact`
- 50+ React controls
- Webpack integration required
- Official Storybook documentation

**SSE vs WebSocket**:
- SSE: server-to-client, automatic reconnection, simpler setup
- WebSocket: bidirectional, requires custom reconnection
- Recommendation: SSE for run-agent serve (server-push-only scenario)

**Message Bus Patterns**:
- Append-only immutability
- Event ordering critical
- Multi-threaded consistency concerns
- Go channels for internal distribution

**Go Process Management**:
- Use process groups (PGID) for hierarchies
- Graceful shutdown: SIGTERM → 30s → SIGKILL
- Find and terminate child PIDs recursively
- `goprocess` package for complex orchestration

### 3. Updated Core Documents

#### SUBSYSTEMS.md
- Added references to new schema specification docs
- Enhanced descriptions with detailed scope information
- Added "Additional Planning Documents" section
- All subsystems now reference complete specifications

#### TOPICS.md (renamed from TIPICS.md)
- Corrected filename typo
- All topics now reference latest decisions
- Cross-references to new schema docs maintained

#### MESSAGE-BUS.md
- Added Round 6 progress entry (2026-02-04T17:31:55Z)
- Documented all consolidated changes and new specifications

#### ISSUES.md
- Added xAI integration to backlog section

### 4. Cleaned Up Question Files

Updated all `*-QUESTIONS.md` files:
- Removed answered questions
- Most files now state "No open questions at this time"
- Remaining questions are actionable experiments needed for agent backends

Files with no open questions:
- subsystem-storage-layout-QUESTIONS.md
- subsystem-env-contract-QUESTIONS.md
- subsystem-message-bus-tools-QUESTIONS.md
- subsystem-message-bus-object-model-QUESTIONS.md
- subsystem-agent-protocol-QUESTIONS.md
- subsystem-monitoring-ui-QUESTIONS.md
- subsystem-runner-orchestration-QUESTIONS.md
- subsystem-agent-backend-xai-QUESTIONS.md

### 5. Verification Process Started

Created `prompts/verification-prompt.md` with comprehensive verification criteria:
- Completeness, consistency, correctness checks
- Cross-reference validation
- Technical decision soundness
- Structured output format

Started Gemini sub-agent for specification verification (runs7).

## Key Decisions Documented

1. **UTF-8 Encoding**: Strict UTF-8 without BOM for all text files
2. **Schema Versioning**: run-info.yaml includes version field (v1)
3. **Signal Handling**: SIGTERM with 30s grace period before SIGKILL
4. **Environment**: Full environment inheritance (no sandbox in MVP)
5. **Path Normalization**: OS-native using Go filepath.Clean
6. **Message Bus**: msg_id printed to stdout on post
7. **Cross-Scope References**: Task messages can reference project messages
8. **UI Streaming**: Single SSE endpoint for all file streaming
9. **State Management**: React Context + hooks (no Redux)
10. **Config Validation**: Embedded schema with `run-agent config` commands
11. **Perplexity Output**: Uses stdout, no output.md
12. **xAI Integration**: Deferred to post-MVP

## Implementation-Ready Specifications

The following subsystems now have complete, implementation-ready specifications:

1. ✅ Storage & Data Layout (with run-info.yaml schema)
2. ✅ Environment & Invocation Contract
3. ✅ Message Bus Tooling & Object Model
4. ✅ Runner & Orchestration (with config.hcl schema)
5. ✅ Agent Protocol & Governance
6. ⚠️ Monitoring UI (needs API endpoint schemas - documented in TOPICS.md)
7. ⚠️ Agent Backends (need CLI flag experimentation for claude/codex/gemini)

## Technical Debt / Next Steps

1. **Agent Backend Experimentation**: Run experiments with claude/codex/gemini CLIs to determine:
   - Correct CLI flags for tool enablement
   - Environment variable mappings
   - Streaming behavior support

2. **API Contract Definition**: Define exact REST endpoint schemas for monitoring UI:
   - Request/response formats
   - TypeScript type generation approach
   - Integration test structure

3. **Perplexity Streaming**: Research Perplexity API streaming capabilities

4. **Sub-Agent Verification**: Review verification output from Gemini agent and address any issues

5. **Post-MVP**: xAI backend integration (tracked in ISSUES.md backlog)

## Files Created/Modified

### Created:
- subsystem-storage-layout-run-info-schema.md (2.5KB specification)
- subsystem-runner-orchestration-config-schema.md (6.3KB specification)
- RESEARCH-FINDINGS.md (5.8KB research summary)
- ROUND-6-SUMMARY.md (this file)
- prompts/verification-prompt.md

### Renamed:
- TIPICS.md → TOPICS.md

### Modified:
- subsystem-storage-layout.md (added UTF-8 + schema references)
- subsystem-env-contract.md (added normalization + signal handling)
- subsystem-message-bus-tools.md (added msg_id return value)
- subsystem-message-bus-object-model.md (added cross-scope references)
- subsystem-monitoring-ui.md (added dev workflow + API details)
- subsystem-runner-orchestration.md (added config validation)
- subsystem-agent-protocol.md (already had resolved decisions)
- subsystem-agent-backend-perplexity.md (changed output convention)
- SUBSYSTEMS.md (enhanced all entries)
- MESSAGE-BUS.md (added progress entry)
- ISSUES.md (added xAI to backlog)
- All 13 *-QUESTIONS.md files (cleaned up answered questions)

## Statistics

- **Total Specification Files**: 13 subsystem specs + 2 schema specs = 15
- **Questions Resolved**: ~25 major questions integrated into specs
- **Questions Remaining**: 4 (all requiring CLI experimentation)
- **Research Topics Covered**: 5 (HCL, Ring UI, SSE/WS, Message Bus, Process Mgmt)
- **Lines of Specification**: ~1200 lines across all docs
- **Schema Fields Defined**: 21 required + 4 optional in run-info.yaml; 30+ in config.hcl

## Quality Assessment

### Strengths:
- Comprehensive schema definitions with validation rules
- Clear separation of concerns across subsystems
- Well-researched technical decisions
- Implementation-ready specifications for 5/7 subsystems
- Consistent cross-referencing between documents
- Future-proofed with versioning strategies

### Remaining Work:
- Agent backend CLI flag experimentation (practical testing needed)
- REST API endpoint schema definition (TypeScript types + OpenAPI consideration)
- Perplexity streaming research
- Sub-agent verification review

## Conclusion

Round 6 successfully consolidated all planning work into coherent, implementation-ready specifications. The system design is now mature enough to begin Go implementation for the core runner and storage subsystems. Agent backend integrations and monitoring UI will require additional experimentation and API contract definition, but the architectural foundation is solid.

All major design questions have been resolved and documented. The remaining open questions are practical implementation details that require hands-on testing with the actual agent CLIs.
