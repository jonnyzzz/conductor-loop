#!/bin/bash
# Run only Stage 5 Phase 5b (Performance and Acceptance Tests)
set -euo pipefail

# Source the main script to get all functions
source "$(dirname "$0")/run-all-tasks.sh"

log "======================================================================"
log "RUNNING STAGE 5 PHASE 5B ONLY (PERFORMANCE AND ACCEPTANCE)"
log "======================================================================"

# Create prompts for Phase 5b
create_prompt_test_performance
create_prompt_test_acceptance

# Run Phase 5b tasks sequentially
log "Phase 5b: Performance and Acceptance Tests (sequential)"

log "Step 5.4: Performance Tests"
run_agent_task "test-performance" "codex" "$PROMPTS_DIR/test-performance.md"
wait_for_tasks "test-performance"

if ! check_task_success "test-performance"; then
    log_error "Phase 5b failed: test-performance"
    exit 1
fi

log "Step 5.5: Acceptance Tests"
run_agent_task "test-acceptance" "codex" "$PROMPTS_DIR/test-acceptance.md"
wait_for_tasks "test-acceptance"

if ! check_task_success "test-acceptance"; then
    log_error "Phase 5b failed: test-acceptance"
    exit 1
fi

log "======================================================================"
log_success "PHASE 5B COMPLETE"
log "======================================================================"
log_success "STAGE 5 COMPLETE (with Phase 5a partial: test-unit timeout accepted)"
log "======================================================================"
