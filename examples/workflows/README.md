# Workflow Templates

Reusable workflow templates for common use cases. Each template includes a prompt and example usage.

## Available Workflows

| Workflow | Use Case | Description |
|----------|----------|-------------|
| [code-review.md](#code-review) | Code quality | Comprehensive code review with multiple agents |
| [documentation.md](#documentation) | Docs generation | Generate complete documentation for codebase |
| [testing.md](#testing) | Test generation | Create comprehensive test suites |
| [refactoring.md](#refactoring) | Code improvement | Systematic refactoring workflow |
| [security-audit.md](#security-audit) | Security | Multi-agent security analysis |
| [performance-optimization.md](#performance-optimization) | Performance | Identify and fix performance issues |

## Usage

Each workflow template can be used directly or customized for your needs:

```bash
run-agent server job submit \
  --project-id my-project \
  --task-id code-review \
  --agent claude \
  --prompt-file workflows/code-review.md
```

## Templates

### code-review.md

**Purpose:** Comprehensive code review using multiple agents for different perspectives.

**Agents recommended:** Claude (analysis), Codex (practical fixes), Gemini (patterns)

**Workflow:**
1. Structural analysis
2. Security review
3. Performance analysis
4. Style and best practices
5. Aggregated report

### documentation.md

**Purpose:** Generate complete API documentation, user guides, and inline comments.

**Agents recommended:** Claude (explanations), Codex (code examples)

**Workflow:**
1. Extract API surface
2. Generate API docs
3. Create usage examples
4. Write user guide
5. Add inline comments

### testing.md

**Purpose:** Generate comprehensive test suites including unit, integration, and edge cases.

**Agents recommended:** Codex (test generation), Claude (edge cases)

**Workflow:**
1. Analyze code coverage
2. Generate unit tests
3. Generate integration tests
4. Add edge case tests
5. Create test documentation

### refactoring.md

**Purpose:** Systematic refactoring with safety checks and validation.

**Agents recommended:** Codex (refactoring), Claude (design review)

**Workflow:**
1. Identify refactoring opportunities
2. Propose refactoring plan
3. Execute refactoring
4. Run tests
5. Validate functionality

### security-audit.md

**Purpose:** Multi-layered security analysis with different agent perspectives.

**Agents recommended:** Claude, Codex, Gemini (consensus)

**Workflow:**
1. Vulnerability scan
2. Authentication/authorization review
3. Input validation check
4. Dependency audit
5. Security report

### performance-optimization.md

**Purpose:** Identify bottlenecks and optimize performance systematically.

**Agents recommended:** Codex (profiling), Claude (algorithms)

**Workflow:**
1. Profile current performance
2. Identify bottlenecks
3. Propose optimizations
4. Implement changes
5. Benchmark results

## Customization

To customize a workflow:

1. Copy the template
2. Modify the prompt for your specific needs
3. Adjust agent selection
4. Add project-specific context
5. Test with a small sample first

## Best Practices

- **Start small:** Test workflows on small codebases first
- **Iterate:** Refine prompts based on results
- **Combine workflows:** Chain workflows for complex tasks
- **Use appropriate agents:** Match agent strengths to tasks
- **Version control:** Track workflow prompt versions
- **Document customizations:** Note changes from templates

## See Also

- [Examples](../) - Working examples
- [Patterns](../patterns.md) - Architectural patterns
- [Best Practices](../best-practices.md) - Production guidelines
