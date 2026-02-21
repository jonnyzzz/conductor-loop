# Conductor Loop - Common Patterns

Reusable architectural patterns for agent orchestration workflows.

## Table of Contents

1. [Fan-Out Pattern](#fan-out-pattern)
2. [Sequential Pipeline](#sequential-pipeline)
3. [Map-Reduce Pattern](#map-reduce-pattern)
4. [Retry with Exponential Backoff](#retry-with-exponential-backoff)
5. [Health Monitoring Pattern](#health-monitoring-pattern)
6. [Rolling Deployment Pattern](#rolling-deployment-pattern)
7. [Parallel Comparison Pattern](#parallel-comparison-pattern)
8. [Hierarchical Decomposition](#hierarchical-decomposition)
9. [Event-Driven Workflow](#event-driven-workflow)
10. [Checkpoint and Resume](#checkpoint-and-resume)

---

## Fan-Out Pattern

**Problem:** Process multiple independent items in parallel.

**Solution:** Parent spawns N children, waits for all, aggregates results.

### Architecture

```
Parent Task
├── Child 1 (item A)
├── Child 2 (item B)
├── Child 3 (item C)
└── Child N (item N)
↓
Parent aggregates results
```

### Implementation

**parent-prompt.md:**
```bash
#!/bin/bash

# Get list of items to process
ITEMS=(item1.txt item2.txt item3.txt item4.txt)

# Spawn child for each item
for item in "${ITEMS[@]}"; do
    run-agent task \
        --project-id fanout-demo \
        --task-id "process-$item" \
        --agent codex \
        --prompt "process $item and write to output.md" &
done

# Wait for all children
wait

# Aggregate results
echo "# Aggregated Results" > output.md
for item in "${ITEMS[@]}"; do
    echo "## $item" >> output.md
    cat "runs/fanout-demo/process-$item/*/output.md" >> output.md
done
```

### Use Cases

- Processing multiple files
- Multi-target deployment
- Parallel testing
- Batch operations

### Advantages

- Maximum parallelism
- Independent failure isolation
- Linear scaling with resources

### Considerations

- Memory usage (N concurrent agents)
- Coordination overhead
- Aggregation complexity
- Error handling for partial failures

---

## Sequential Pipeline

**Problem:** Multi-stage processing where each stage depends on the previous.

**Solution:** Chain tasks where output of stage N becomes input to stage N+1.

### Architecture

```
Stage 1 → Stage 2 → Stage 3 → Stage 4
(Input)   (Transform) (Validate) (Output)
```

### Implementation

**pipeline-orchestrator.md:**
```bash
#!/bin/bash
set -e  # Exit on any failure

PROJECT_ID="pipeline-demo"

# Stage 1: Extract
echo "Stage 1: Extracting data..."
run-agent task \
    --project-id $PROJECT_ID \
    --task-id stage1-extract \
    --agent codex \
    --prompt-file prompts/extract.md

# Get stage 1 output
STAGE1_OUTPUT=$(cat runs/$PROJECT_ID/stage1-extract/*/output.md)

# Stage 2: Transform
echo "Stage 2: Transforming data..."
run-agent task \
    --project-id $PROJECT_ID \
    --task-id stage2-transform \
    --agent codex \
    --prompt "Transform this data: $STAGE1_OUTPUT"

# Stage 3: Validate
echo "Stage 3: Validating results..."
run-agent task \
    --project-id $PROJECT_ID \
    --task-id stage3-validate \
    --agent claude \
    --prompt-file prompts/validate.md

# Stage 4: Publish
echo "Stage 4: Publishing..."
run-agent task \
    --project-id $PROJECT_ID \
    --task-id stage4-publish \
    --agent codex \
    --prompt-file prompts/publish.md

echo "Pipeline complete!"
```

### Use Cases

- ETL workflows
- Code review → fix → test → deploy
- Data preprocessing pipelines
- Multi-stage validation

### Advantages

- Clear dependencies
- Easy to debug (isolate stage)
- Natural checkpoints
- Incremental progress

### Considerations

- No parallelism (sequential)
- Failure stops entire pipeline
- Longer total duration
- State passing between stages

---

## Map-Reduce Pattern

**Problem:** Process large dataset in parallel, then combine results.

**Solution:** Map phase distributes work, Reduce phase aggregates.

### Architecture

```
          Map Phase (parallel)
Input → [Worker 1, Worker 2, ..., Worker N]
          ↓
        Reduce Phase
        [Combiner] → Final Result
```

### Implementation

**map-reduce-orchestrator.md:**
```bash
#!/bin/bash
set -e

PROJECT_ID="mapreduce-demo"

# MAP PHASE
echo "Map phase: Processing chunks..."

CHUNKS=(chunk1 chunk2 chunk3 chunk4)
for chunk in "${CHUNKS[@]}"; do
    run-agent task \
        --project-id $PROJECT_ID \
        --task-id "map-$chunk" \
        --agent codex \
        --prompt "Count words in $chunk, write JSON to output.md" &
done

wait
echo "Map phase complete."

# REDUCE PHASE
echo "Reduce phase: Aggregating results..."

# Collect all map outputs
MAP_RESULTS=""
for chunk in "${CHUNKS[@]}"; do
    RESULT=$(cat runs/$PROJECT_ID/map-$chunk/*/output.md)
    MAP_RESULTS="$MAP_RESULTS\n$RESULT"
done

# Run reducer
run-agent task \
    --project-id $PROJECT_ID \
    --task-id reduce-aggregate \
    --agent codex \
    --prompt "Aggregate these word counts into final totals: $MAP_RESULTS"

# Final output
cp runs/$PROJECT_ID/reduce-aggregate/*/output.md final-results.md
echo "MapReduce complete! Results in final-results.md"
```

### Use Cases

- Large file processing
- Log analysis
- Distributed computation
- Batch analytics

### Advantages

- Scalable to large datasets
- Parallelism in map phase
- Fault tolerance (retry individual mappers)
- Natural sharding

### Considerations

- Overhead for small datasets
- Reduce phase is bottleneck
- Data partitioning strategy
- Shuffle phase complexity (if needed)

---

## Retry with Exponential Backoff

**Problem:** Transient failures should be retried without overwhelming system.

**Solution:** Retry with increasing delays: 1s, 2s, 4s, 8s, 16s...

### Implementation

```bash
#!/bin/bash

MAX_RETRIES=5
RETRY_COUNT=0
BASE_DELAY=1

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    echo "Attempt $((RETRY_COUNT + 1))/$MAX_RETRIES"

    if run-agent task \
        --project-id retry-demo \
        --task-id flaky-task \
        --agent codex \
        --prompt-file prompts/task.md; then
        echo "Success!"
        exit 0
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))

    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        DELAY=$((BASE_DELAY * (2 ** RETRY_COUNT)))
        echo "Failed. Retrying in ${DELAY}s..."
        sleep $DELAY
    fi
done

echo "Failed after $MAX_RETRIES attempts"
exit 1
```

### Use Cases

- Network requests
- API rate limits
- Transient errors
- Resource contention

### Advantages

- Handles transient failures
- Reduces retry storms
- Automatic recovery
- Fair resource usage

### Considerations

- Total time increases
- Not for permanent failures
- Jitter for thundering herd
- Max retry limits

### With Jitter

```bash
# Add randomness to avoid synchronized retries
JITTER=$((RANDOM % DELAY))
sleep $((DELAY + JITTER))
```

---

## Health Monitoring Pattern

**Problem:** Detect and alert on unhealthy system state.

**Solution:** Periodic health checks with automated remediation.

### Architecture

```
Health Monitor (every 60s)
├── Check API health
├── Check agent connectivity
├── Check disk space
├── Check queue depth
└── Check error rate
    ↓
Alerts + Auto-remediation
```

### Implementation

**health-monitor.sh:**
```bash
#!/bin/bash

while true; do
    echo "[$(date)] Running health checks..."

    # API health
    if ! curl -sf http://localhost:14355/api/v1/health > /dev/null; then
        echo "ALERT: API unhealthy"
        # Send alert
        # Attempt restart
    fi

    # Disk space
    DISK_USAGE=$(df /data/runs | tail -1 | awk '{print $5}' | sed 's/%//')
    if [ $DISK_USAGE -gt 85 ]; then
        echo "ALERT: Disk usage at ${DISK_USAGE}%"
        # Cleanup old runs
        find /data/runs -mtime +30 -delete
    fi

    # Queue depth
    QUEUE_DEPTH=$(conductor task list --status pending | wc -l)
    if [ $QUEUE_DEPTH -gt 100 ]; then
        echo "ALERT: Queue depth at $QUEUE_DEPTH"
        # Scale up workers
    fi

    # Error rate
    ERROR_COUNT=$(find runs -name "run-info.yaml" -mmin -60 -exec grep -l "status: failed" {} \; | wc -l)
    TOTAL_COUNT=$(find runs -name "run-info.yaml" -mmin -60 | wc -l)
    if [ $TOTAL_COUNT -gt 0 ]; then
        ERROR_RATE=$((ERROR_COUNT * 100 / TOTAL_COUNT))
        if [ $ERROR_RATE -gt 20 ]; then
            echo "ALERT: Error rate at ${ERROR_RATE}%"
            # Investigate and alert
        fi
    fi

    sleep 60
done
```

### Use Cases

- Production monitoring
- SLA compliance
- Automated remediation
- Capacity planning

---

## Rolling Deployment Pattern

**Problem:** Deploy new version without downtime.

**Solution:** Gradually replace instances with new version.

### Architecture

```
Old: [V1] [V1] [V1] [V1]
     ↓
Mix: [V2] [V1] [V1] [V1]  (25%)
     ↓
Mix: [V2] [V2] [V1] [V1]  (50%)
     ↓
Mix: [V2] [V2] [V2] [V1]  (75%)
     ↓
New: [V2] [V2] [V2] [V2]  (100%)
```

### Implementation

**rolling-deploy.sh:**
```bash
#!/bin/bash

NEW_VERSION="v2.0"
INSTANCES=(conductor-1 conductor-2 conductor-3 conductor-4)

for instance in "${INSTANCES[@]}"; do
    echo "Deploying $NEW_VERSION to $instance..."

    # Deploy new version
    ssh $instance "docker pull conductor:$NEW_VERSION"
    ssh $instance "docker-compose up -d"

    # Wait for health check
    for i in {1..30}; do
        if curl -sf "http://$instance:14355/api/v1/health"; then
            echo "$instance healthy"
            break
        fi
        sleep 2
    done

    # Smoke test
    if ! run-agent task \
        --project-id deploy-test \
        --task-id smoke-$instance \
        --agent codex \
        --prompt "echo success"; then
        echo "Smoke test failed! Rolling back $instance"
        ssh $instance "docker-compose down && docker-compose -f docker-compose.old.yml up -d"
        exit 1
    fi

    echo "$instance deployed successfully"
    sleep 30  # Bake time
done

echo "Rolling deployment complete!"
```

---

## Parallel Comparison Pattern

**Problem:** Get multiple perspectives on the same problem.

**Solution:** Run multiple agents in parallel, compare results.

### Implementation

```bash
#!/bin/bash

PROJECT_ID="comparison"
PROMPT_FILE="prompts/analyze.md"
AGENTS=(claude codex gemini)

# Run all agents in parallel
for agent in "${AGENTS[@]}"; do
    run-agent task \
        --project-id $PROJECT_ID \
        --task-id "analyze-$agent" \
        --agent $agent \
        --prompt-file $PROMPT_FILE &
done

wait

# Compare results
echo "# Multi-Agent Comparison" > comparison.md
for agent in "${AGENTS[@]}"; do
    echo "## $agent Analysis" >> comparison.md
    cat "runs/$PROJECT_ID/analyze-$agent/*/output.md" >> comparison.md
    echo "" >> comparison.md
done

# Optional: Consensus task
run-agent task \
    --project-id $PROJECT_ID \
    --task-id consensus \
    --agent claude \
    --prompt "Find common themes in comparison.md"
```

### Use Cases

- Code review
- Security audit
- Decision making
- Quality assurance

---

## Hierarchical Decomposition

**Problem:** Complex task needs recursive breakdown.

**Solution:** Parent spawns children, children spawn grandchildren.

### Architecture

```
Root (Orchestrator)
├── Module A
│   ├── Unit A.1
│   └── Unit A.2
├── Module B
│   ├── Unit B.1
│   ├── Unit B.2
│   └── Unit B.3
└── Module C
    └── Unit C.1
```

### Implementation

**Level 1: Root**
```bash
for module in module-a module-b module-c; do
    run-agent task \
        --task-id $module \
        --prompt "Decompose $module into units and process each" &
done
wait
```

**Level 2: Module (auto-generated prompt)**
```bash
for unit in unit-1 unit-2 unit-3; do
    run-agent task \
        --task-id "$MODULE_NAME-$unit" \
        --prompt "Process $unit" &
done
wait
```

### Limit Depth

```bash
# Maximum recursion depth: 16 (configurable)
if [ $CURRENT_DEPTH -ge $MAX_DEPTH ]; then
    echo "Maximum depth reached"
    exit 1
fi
```

---

## Event-Driven Workflow

**Problem:** React to events as they occur.

**Solution:** Monitor message bus, trigger tasks based on events.

### Implementation

**event-dispatcher.sh:**
```bash
#!/bin/bash

LAST_MSG_ID=""

while true; do
    # Read new messages since last check
    MESSAGES=$(messagebus-tool read --since-id "$LAST_MSG_ID")

    echo "$MESSAGES" | while read -r msg; do
        MSG_TYPE=$(echo "$msg" | grep "^type:" | awk '{print $2}')
        MSG_BODY=$(echo "$msg" | grep "^body:")

        case $MSG_TYPE in
            "CODE_PUSHED")
                # Trigger CI pipeline
                run-agent task --task-id ci-run-$TIMESTAMP ... &
                ;;
            "TEST_FAILED")
                # Trigger investigation
                run-agent task --task-id investigate-$TIMESTAMP ... &
                ;;
            "DEPLOY_REQUESTED")
                # Trigger deployment
                run-agent task --task-id deploy-$TIMESTAMP ... &
                ;;
        esac

        LAST_MSG_ID=$(echo "$msg" | grep "^msg_id:" | awk '{print $2}')
    done

    sleep 5  # Poll interval
done
```

---

## Checkpoint and Resume

**Problem:** Long-running task needs to resume after interruption.

**Solution:** Save progress, resume from checkpoint.

### Implementation

```bash
#!/bin/bash

CHECKPOINT_FILE="checkpoint.json"

# Resume from checkpoint if exists
if [ -f "$CHECKPOINT_FILE" ]; then
    echo "Resuming from checkpoint..."
    COMPLETED=$(jq -r '.completed[]' "$CHECKPOINT_FILE")
    CURRENT_STEP=$(jq -r '.current_step' "$CHECKPOINT_FILE")
else
    COMPLETED=()
    CURRENT_STEP=0
fi

STEPS=(step1 step2 step3 step4 step5)

for i in "${!STEPS[@]}"; do
    if [ $i -lt $CURRENT_STEP ]; then
        echo "Skipping ${STEPS[$i]} (already completed)"
        continue
    fi

    echo "Processing ${STEPS[$i]}..."

    if run-agent task --task-id "${STEPS[$i]}" ...; then
        # Update checkpoint
        jq ".completed += [\"${STEPS[$i]}\"] | .current_step = $((i + 1))" \
            "$CHECKPOINT_FILE" > "$CHECKPOINT_FILE.tmp"
        mv "$CHECKPOINT_FILE.tmp" "$CHECKPOINT_FILE"
    else
        echo "Failed at ${STEPS[$i]}"
        exit 1
    fi
done

rm "$CHECKPOINT_FILE"
echo "All steps complete!"
```

---

## Pattern Selection Guide

| Pattern | Parallelism | Complexity | Use Case |
|---------|-------------|------------|----------|
| Fan-Out | High | Low | Independent batch processing |
| Sequential Pipeline | None | Low | Dependent stages |
| Map-Reduce | High | Medium | Large dataset processing |
| Retry + Backoff | N/A | Low | Fault tolerance |
| Health Monitoring | N/A | Medium | Production monitoring |
| Rolling Deployment | Medium | High | Zero-downtime deploys |
| Parallel Comparison | High | Low | Multi-perspective analysis |
| Hierarchical | High | High | Complex decomposition |
| Event-Driven | High | High | Reactive workflows |
| Checkpoint/Resume | N/A | Medium | Long-running tasks |

---

## Combining Patterns

Patterns can be combined for sophisticated workflows:

**Example: CI/CD Pipeline**
```
Sequential Pipeline:
├── Fan-Out: Run tests in parallel
├── Parallel Comparison: Multiple security scanners
├── Rolling Deployment: Gradual rollout
└── Health Monitoring: Post-deploy validation
```

**Example: Data Processing**
```
Map-Reduce:
├── Map: Fan-Out pattern with retry
├── Reduce: Sequential Pipeline
└── Checkpoint: Save progress throughout
```

---

## See Also

- [Best Practices](./best-practices.md) - Production guidelines
- [Examples](./examples/) - Working implementations
- [Configuration Templates](./configs/) - Ready-to-use configs
