#!/usr/bin/env bash
# mv /usr/local/bin/helm /usr/local/bin/helm.real
# ln -s /path/scripts/helm-wrapper.sh /usr/local/bin/helm

AUDITOR_API="${AUDITOR_API:-http://localhost:8080}"
AUDITOR_TOKEN="${AUDITOR_TOKEN:-}"
HELM_BIN="${HELM_BIN:-/usr/local/bin/helm.real}"

user="${USER:-unknown}"
host="$(hostname -f 2>/dev/null || hostname)"
cmd="helm $*"

payload=$(jq -c --arg user "$user" --arg host "$host" --arg cmd "$cmd" \
  '{user:$user,host:$host,source:"helm",command:$cmd}')
auth=()
[ -n "$AUDITOR_TOKEN" ] && auth=(-H "Authorization: Bearer $AUDITOR_TOKEN")
curl -fsS -X POST "$AUDITOR_API/v1/events" -H 'Content-Type: application/json' "${auth[@]}" -d "$payload" >/dev/null 2>&1 || true

exec "$HELM_BIN" "$@"