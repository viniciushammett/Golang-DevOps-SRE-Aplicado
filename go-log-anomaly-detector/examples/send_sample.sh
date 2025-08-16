#!/usr/bin/env bash
set -euo pipefail

API="${API:-http://localhost:8080}"
AUTH="${AUTH:-}" # ex: "Bearer token123"

send(){ curl -sS -XPOST "$API/v1/logs" -H 'Content-Type: application/json' ${AUTH:+-H "Authorization: $AUTH"} -d "$1" >/dev/null; }

for i in $(seq 1 25); do
  send '{"source":"app","msg":"GET /api/orders -> HTTP/1.1\" 500"}'
done

for i in $(seq 1 12); do
  send '{"source":"auth","msg":"login failed for user=alice ip=1.2.3.4"}'
done

echo "ok; veja /v1/anomalies"