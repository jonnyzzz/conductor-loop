# Problem: Perplexity Output Double-Write Unclear

## Context
From subsystem-agent-backend-perplexity.md:
> "Writes BOTH stdout AND output.md"

Other backends: Only write to stdout, runner creates output.md

## Problem
1. Sequential or parallel writes?
2. What if streaming to stdout fails mid-stream but output.md succeeds?
3. Why is Perplexity different from other backends?

## Your Task
Decide:
1. **Unify**: Make Perplexity write only to stdout (like others)
2. **Document**: Keep double-write but specify exact behavior
3. **Generalize**: Make ALL backends write both

Specify exact write order and error handling.
