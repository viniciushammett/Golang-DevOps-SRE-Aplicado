#!/usr/bin/env bash
set -euo pipefail
CLUSTER_NAME="monitoring-lab"

echo ">> Deletando cluster kind '${CLUSTER_NAME}'..."
kind delete cluster --name "${CLUSTER_NAME}" || true
echo "✅ Ambiente removido."