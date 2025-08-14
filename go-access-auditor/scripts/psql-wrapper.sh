#!/usr/bin/env bash
# mv /usr/bin/psql /usr/bin/psql.real
# ln -s /path/scripts/psql-wrapper.sh /usr/bin/psql

AUDITOR_API="${AUDITOR_API:-http://localhost:8080}"
AUDITOR_TOKEN="${AUDITOR_TOKEN:-}"
PSQL_BIN="${PSQL_BIN:-/usr/bin/psql.real}"

user="${USER:-unknown}"
host="$(hostname -f 2>/dev/null || hostname)"
cmd="psql $*"

payload=$(jq -c --arg user "$user" --arg host "$host" --arg cmd "$cmd" \
  '{user:$user,host:$host,source:"psql",command:$cmd}')
auth=()
[ -n "$AUDITOR_TOKEN" ] && auth=(-H "Authorization: Bearer $AUDITOR_TOKEN")
curl -fsS -X POST "$AUDITOR_API/v1/events" -H 'Content-Type: application/json' "${auth[@]}" -d "$payload" >/dev/null 2>&1 || true

exec "$PSQL_BIN" "$@"