# Task: Create Documentation Examples

**Task ID**: docs-examples
**Phase**: Documentation
**Agent Type**: Documentation (Claude preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-5 complete

## Objective
Create practical examples, templates, and tutorial projects that demonstrate real-world usage of Conductor Loop.

## Required Examples

### 1. examples/README.md
**Examples Overview**:
- List all examples
- What each example demonstrates
- How to run each example
- Prerequisites for each

### 2. examples/hello-world/
**Simple Hello World Example**:
```
examples/hello-world/
├── README.md
├── config.yaml
├── prompt.md
└── run.sh
```

Demonstrates:
- Basic task execution
- Single agent (Codex)
- Simple prompt
- Viewing results

### 3. examples/multi-agent/
**Multi-Agent Comparison**:
Run the same task with different agents (Claude, Codex, Gemini) and compare results.

Demonstrates:
- Running multiple agents
- Comparing outputs
- Agent-specific behavior

### 4. examples/parent-child/
**Parent-Child Task Hierarchy**:
Parent task spawns 3 child tasks, each doing different work.

Demonstrates:
- run-agent task command
- Parent-child relationships
- Run tree visualization
- Waiting for children

### 5. examples/ralph-loop/
**Ralph Loop Wait Pattern**:
Task creates DONE file but has long-running children.

Demonstrates:
- DONE file usage
- Wait-without-restart
- Child process management
- Ralph Loop behavior

### 6. examples/message-bus/
**Message Bus Communication**:
Multiple agents writing to message bus concurrently.

Demonstrates:
- Message bus usage
- Inter-agent communication
- Race-free concurrent writes
- Message ordering

### 7. examples/rest-api/
**REST API Usage**:
Scripts showing all API endpoints with curl examples.

Demonstrates:
- Creating tasks via API
- Polling for completion
- Streaming logs via SSE
- Error handling

### 8. examples/web-ui-demo/
**Web UI Demo Scenario**:
Long-running task with progress updates visible in UI.

Demonstrates:
- Real-time log streaming
- Status updates
- UI features
- Live monitoring

### 9. examples/docker-deployment/
**Docker Deployment**:
Complete Docker setup for production deployment.

Files:
- docker-compose.yml (production-ready)
- config.yaml (production config)
- nginx.conf (reverse proxy)
- README.md (deployment guide)

Demonstrates:
- Docker deployment
- Reverse proxy setup
- Environment variables
- Production configuration
- Health checks

### 10. examples/ci-integration/
**CI/CD Integration**:
GitHub Actions workflow using Conductor Loop.

Demonstrates:
- CI/CD usage
- Automated testing
- Multi-agent validation
- Result aggregation

### 11. examples/custom-agent/
**Custom Agent Backend**:
Template for implementing a custom agent.

Demonstrates:
- Agent interface implementation
- Configuration
- Integration testing
- Registration

### 12. Configuration Templates

Create templates in examples/configs/:
- config.basic.yaml (minimal config)
- config.production.yaml (production-ready)
- config.multi-agent.yaml (all agents configured)
- config.docker.yaml (Docker-optimized)
- config.development.yaml (dev environment)

### 13. Workflow Templates

Create workflow templates in examples/workflows/:
- code-review.md (use Claude for code review)
- documentation.md (generate docs with agents)
- testing.md (run tests with multiple agents)
- refactoring.md (automated refactoring workflow)

### 14. Tutorial Project

Create examples/tutorial/:
A complete step-by-step tutorial that builds a real project using Conductor Loop.

**Tutorial: Building a Multi-Agent Code Analyzer**

Steps:
1. Setup and installation
2. Create first task (analyze single file)
3. Add parent task (analyze multiple files)
4. Compare agent results
5. Aggregate findings
6. Generate report
7. View in Web UI

Each step has:
- Clear instructions
- Working code
- Expected output
- Troubleshooting tips

### 15. Best Practices Guide

Create docs/examples/best-practices.md:
- Task design patterns
- Prompt engineering tips
- Error handling strategies
- Performance optimization
- Security considerations
- Production deployment checklist

### 16. Common Patterns

Create docs/examples/patterns.md:
- Fan-out pattern (1 parent, N children)
- Sequential pipeline (task1 → task2 → task3)
- Map-reduce pattern
- Retry with exponential backoff
- Health monitoring pattern
- Rolling deployment pattern

## Example Standards

**All examples must**:
- Be self-contained and runnable
- Include clear README with instructions
- Show expected output
- Include error handling
- Be tested and verified
- Have inline comments explaining key parts

**File structure**:
```
examples/example-name/
├── README.md          # What it does, how to run
├── config.yaml        # Configuration
├── prompt.md          # Task prompt (if applicable)
├── run.sh            # Script to run the example
└── expected-output/   # What success looks like
```

## Success Criteria
- All examples working and tested
- Configuration templates provided
- Tutorial project complete
- Best practices documented
- Common patterns explained
- Examples cover all major features
- New users can learn from examples

## Output
Log to MESSAGE-BUS.md:
- FACT: All examples created and tested
- FACT: Configuration templates complete
- FACT: Tutorial project working
- FACT: Best practices guide written
- FACT: Common patterns documented
