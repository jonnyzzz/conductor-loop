#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <run_id>"
    exit 1
fi

RUN_ID=$1
API_BASE="http://localhost:8080/api/v1"

echo "Polling status for run: $RUN_ID"
echo ""

while true; do
    RESPONSE=$(curl -s "$API_BASE/runs/$RUN_ID")
    STATUS=$(echo "$RESPONSE" | jq -r '.status')

    echo "[$(date +%H:%M:%S)] Status: $STATUS"

    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
        echo ""
        echo "Final result:"
        echo "$RESPONSE" | jq .
        exit 0
    fi

    sleep 5
done
