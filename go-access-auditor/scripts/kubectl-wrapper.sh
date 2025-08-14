#!/usr/bin/env bash
# mv /usr/local/bin/kubectl /usr/local/bin/kubectl.real
# ln -s /path/scripts/kubectl-wrapper.sh /usr/local/bin/kubectl

AUDITOR_API="${AUDITOR_API:-http://localhost:8080}"
AUDITOR_TOKEN="${AUDITOR_TOKEN:-}"
KUBECTL_BIN="${KUBECTL_BIN:-/usr/local/bin/kubectl.real}"

user="${USER:-unknown}"
host="$(hostname -f 2>/dev/null || hostname)"
cmd="kubectl $*"

payload=$(jq -c --arg user "$user" --arg host "$host" --arg cmd "$cmd" \
  '{user:$user,host:$host,source:"kubectl",command:$cmd}')

auth=()
[ -n "$AUDITOR_TOKEN" ] && auth=(-H "Authorization: Bearer $AUDITOR_TOKEN")
curl -fsS -X POST "$AUDITOR_API/v1/events" -H 'Content-Type: application/json' "${auth[@]}" -d "$payload" >/dev/null 2>&1 || true

exec "$KUBECTL_BIN" "$@"