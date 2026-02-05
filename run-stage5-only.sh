#!/bin/bash
# Run only Stage 5 (Integration and Testing)
set -euo pipefail

# Source the main script to get all functions
source "$(dirname "$0")/run-all-tasks.sh"

# Just run Stage 5
log "======================================================================"
log "RUNNING STAGE 5 ONLY (INTEGRATION AND TESTING)"
log "======================================================================"

# Create prompts for Stage 5
create_prompt_test_unit
create_prompt_test_integration
create_prompt_test_docker
create_prompt_test_performance
create_prompt_test_acceptance

# Run Stage 5
if ! run_stage_5_testing; then
    log_error "Stage 5 (Integration and Testing) failed"
    exit 1
fi

log "======================================================================"
log_success "STAGE 5 COMPLETE"
log "======================================================================"
