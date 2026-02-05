#!/bin/bash
# Simulate high load

for i in {1..100}; do
    curl -X POST http://localhost:8080/api/v1/tasks \
        -H "Content-Type: application/json" \
        -d '{"task_id": "load-'$i'", "agent_type": "codex"}' &
done

wait
echo "Load test complete"
