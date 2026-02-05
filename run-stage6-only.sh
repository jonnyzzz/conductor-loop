#!/bin/bash
# Run only Stage 6 (Documentation)
set -euo pipefail

# Source the main script to get all functions
source "$(dirname "$0")/run-all-tasks.sh"

# Just run Stage 6
log "======================================================================"
log "RUNNING STAGE 6 ONLY (DOCUMENTATION)"
log "======================================================================"

# Create prompts for Stage 6
create_prompt_docs_user
create_prompt_docs_dev
create_prompt_docs_examples

# Run Stage 6
if ! run_stage_6_documentation; then
    log_error "Stage 6 (Documentation) failed"
    exit 1
fi

log "======================================================================"
log_success "STAGE 6 COMPLETE"
log "======================================================================"
log_success "ALL STAGES COMPLETE - PROJECT FINISHED!"
log "======================================================================"
