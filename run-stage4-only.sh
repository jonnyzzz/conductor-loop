#!/bin/bash
# Run only Stage 4 (API and Frontend)
set -euo pipefail

# Source the main script to get all functions
source "$(dirname "$0")/run-all-tasks.sh"

# Just run Stage 4
log "======================================================================"
log "RUNNING STAGE 4 ONLY (API AND FRONTEND)"
log "======================================================================"

# Create prompts for Stage 4
create_prompt_api_rest
create_prompt_api_sse
create_prompt_ui_frontend

# Run Stage 4
if ! run_stage_4_api; then
    log_error "Stage 4 (API and Frontend) failed"
    exit 1
fi

log "======================================================================"
log_success "STAGE 4 COMPLETE"
log "======================================================================"
