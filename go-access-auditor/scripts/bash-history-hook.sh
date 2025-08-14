#!/usr/bin/env bash
# Adicione ao ~/.bashrc ou /etc/profile.d/auditor.sh:
#   source /path/scripts/bash-history-hook.sh

AUDITOR_API="${AUDITOR_API:-http://localhost:8080}"
AUDITOR_TOKEN="${AUDITOR_TOKEN:-}"

__auditor_send() {
  local cmd ts user host src
  cmd="$1"
  ts="$(date -u +%FT%TZ)"
  user="${USER:-unknown}"
  host="$(hostname -f 2>/dev/null || hostname)"
  src="bash"
  payload=$(jq -c --arg when "$ts" --arg user "$user" --arg host "$host" --arg src "$src" --arg cmd "$cmd" \
    '{when:$when,user:$user,host:$host,source:$src,command:$cmd}')
  auth=()
  [ -n "$AUDITOR_TOKEN" ] && auth=(-H "Authorization: Bearer $AUDITOR_TOKEN")
  curl -fsS -X POST "$AUDITOR_API/v1/events" -H 'Content-Type: application/json' "${auth[@]}" -d "$payload" >/dev/null 2>&1 || true
}

# Envia o último comando no PROMPT_COMMAND
__auditor_pc() {
  local last=$(history 1 | sed -E 's/^[[:space:]]*[0-9]+\s+//')
  [ -n "$last" ] && __auditor_send "$last"
}
PROMPT_COMMAND="history -a; __auditor_pc; $PROMPT_COMMAND"