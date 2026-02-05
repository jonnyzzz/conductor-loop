#!/bin/bash

API_BASE="http://localhost:8080/api/v1"

echo "Creating task via API..."

RESPONSE=$(curl -s -X POST "$API_BASE/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "api-demo",
    "task_id": "hello-api",
    "agent_type": "codex",
    "prompt": "Write a simple greeting message to output.md with the current timestamp.",
    "config": {
      "timeout": "300"
    }
  }')

echo "Response:"
echo "$RESPONSE" | jq .

RUN_ID=$(echo "$RESPONSE" | jq -r '.run_id')
echo ""
echo "Created run: $RUN_ID"
echo "Poll status with: ./scripts/03-poll-status.sh $RUN_ID"
