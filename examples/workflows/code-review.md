# Workflow: Comprehensive Code Review

Multi-agent code review workflow that examines code from multiple perspectives.

## Objective

Perform a thorough code review covering:
- Code structure and organization
- Security vulnerabilities
- Performance bottlenecks
- Best practices and style
- Potential bugs

## Recommended Setup

Run with 3 agents in parallel for different perspectives:
- **Claude:** Deep analysis and security
- **Codex:** Practical improvements and fixes
- **Gemini:** Pattern recognition and best practices

## Workflow Steps

### Phase 1: Individual Reviews (Parallel)

**Agent 1 (Claude) - Security & Architecture:**
- Identify security vulnerabilities
- Review authentication/authorization
- Analyze error handling
- Check input validation
- Assess architecture decisions

**Agent 2 (Codex) - Code Quality & Fixes:**
- Find potential bugs
- Suggest code improvements
- Provide concrete code examples
- Check for code smells
- Review testing coverage

**Agent 3 (Gemini) - Patterns & Best Practices:**
- Identify anti-patterns
- Suggest design patterns
- Check coding standards compliance
- Review naming conventions
- Assess maintainability

### Phase 2: Aggregation (Sequential)

**Aggregator Agent:**
- Combine findings from all reviews
- Prioritize issues by severity
- Remove duplicates
- Create actionable recommendations
- Generate executive summary

## Usage

### Option 1: Manual Execution

```bash
# Phase 1: Run reviews in parallel
conductor task create \
  --project-id code-review \
  --task-id review-claude \
  --agent claude \
  --prompt "Review ${TARGET_FILE} for security and architecture. Focus on vulnerabilities, authentication, error handling, and design decisions." &

conductor task create \
  --project-id code-review \
  --task-id review-codex \
  --agent codex \
  --prompt "Review ${TARGET_FILE} for code quality. Find bugs, suggest improvements with code examples, and check test coverage." &

conductor task create \
  --project-id code-review \
  --task-id review-gemini \
  --agent gemini \
  --prompt "Review ${TARGET_FILE} for patterns and best practices. Identify anti-patterns, suggest design patterns, and check coding standards." &

wait

# Phase 2: Aggregate results
conductor task create \
  --project-id code-review \
  --task-id aggregate \
  --agent claude \
  --prompt "Aggregate the three code reviews from runs/code-review/review-*/, prioritize issues, and create a comprehensive report."
```

### Option 2: Automated Script

```bash
#!/bin/bash
# code-review.sh

TARGET_FILE=$1
PROJECT_ID="code-review-$(date +%s)"

echo "Starting multi-agent code review for $TARGET_FILE"

# Launch parallel reviews
echo "Phase 1: Running parallel reviews..."
conductor task create --project-id $PROJECT_ID --task-id claude --agent claude --prompt "Security & Architecture review of $TARGET_FILE" &
conductor task create --project-id $PROJECT_ID --task-id codex --agent codex --prompt "Code quality review of $TARGET_FILE" &
conductor task create --project-id $PROJECT_ID --task-id gemini --agent gemini --prompt "Patterns & best practices review of $TARGET_FILE" &

wait
echo "Phase 1 complete"

# Aggregate
echo "Phase 2: Aggregating results..."
conductor task create --project-id $PROJECT_ID --task-id final --agent claude --prompt "Aggregate reviews from runs/$PROJECT_ID/*/"

echo "Code review complete! Check runs/$PROJECT_ID/final/"
```

## Output Format

The final aggregated review should include:

```markdown
# Code Review: [filename]

## Executive Summary
[High-level assessment of code quality]

## Critical Issues (Must Fix)
1. **[Issue]** - Severity: High
   - Location: file.py:123
   - Description: [What's wrong]
   - Recommendation: [How to fix]
   - Example: [Code snippet]

## Important Issues (Should Fix)
[Same format as critical]

## Suggestions (Nice to Have)
[Same format]

## Positive Aspects
[Things the code does well]

## Overall Recommendations
1. [Prioritized action items]

## Agent Consensus
- All agents agreed on: [issues]
- Conflicting opinions on: [issues with different views]
```

## Customization

### For Different Languages

Modify prompts to include language-specific concerns:

**Python:**
- PEP 8 compliance
- Type hints usage
- Virtual environment handling

**JavaScript:**
- ESLint rules
- Async/await patterns
- Bundle size

**Go:**
- Goroutine safety
- Error handling conventions
- Interface usage

### For Specific Domains

**Web Applications:**
- XSS vulnerabilities
- CSRF protection
- SQL injection
- Session management

**APIs:**
- REST conventions
- Authentication methods
- Rate limiting
- API versioning

**Data Processing:**
- Memory efficiency
- Batch processing
- Error handling
- Data validation

## Success Criteria

A successful code review produces:
- ✓ Actionable recommendations
- ✓ Specific line numbers for issues
- ✓ Code examples for fixes
- ✓ Prioritized by severity
- ✓ Consensus across agents on critical issues

## Tips

1. **Start with small files:** Test the workflow on ~200 line files first
2. **Provide context:** Include README or architecture docs in the prompt
3. **Iterate on prompts:** Refine based on results
4. **Set clear standards:** Reference style guides in prompts
5. **Review the reviews:** Agents can miss things; human review still essential

## Integration with CI/CD

```yaml
# .github/workflows/code-review.yml
name: Agent Code Review

on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run multi-agent review
        run: ./workflows/code-review.sh ${{ github.event.pull_request.changed_files }}
      - name: Post results
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const review = fs.readFileSync('review-output.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: review
            });
```

## Related Workflows

- [security-audit.md](./security-audit.md) - Deep security focus
- [refactoring.md](./refactoring.md) - Post-review improvements
- [testing.md](./testing.md) - Generate tests for found issues
