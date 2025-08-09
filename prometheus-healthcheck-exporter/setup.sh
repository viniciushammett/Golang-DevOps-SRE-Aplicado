#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="monitoring-lab"
NAMESPACE="monitoring-lab"
HELM_RELEASE="lab-prom"
IMAGE_NAME="healthcheck-exporter:latest"

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "Erro: '$1' não encontrado no PATH."; exit 1; }
}

echo ">> Verificando dependências..."
need kind
need kubectl
need helm
need docker

echo ">> Criando cluster kind '${CLUSTER_NAME}' (se não existir)..."
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  kind create cluster --name "${CLUSTER_NAME}" --config kind-cluster.yaml
else
  echo "   Cluster já existe."
fi

echo ">> Instalando kube-prometheus-stack via Helm..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo update >/dev/null
kubectl get ns "${NAMESPACE}" >/dev/null 2>&1 || kubectl create ns "${NAMESPACE}"
helm upgrade --install "${HELM_RELEASE}" prometheus-community/kube-prometheus-stack \
  -n "${NAMESPACE}" --wait --timeout 10m

echo ">> Buildando imagem local do exporter..."
docker build -t "${IMAGE_NAME}" .

echo ">> Carregando imagem no cluster kind..."
kind load docker-image "${IMAGE_NAME}" --name "${CLUSTER_NAME}"

echo ">> Aplicando manifests do exporter..."
kubectl apply -f k8s-exporter-stack.yaml

kubectl -n "${NAMESPACE}" rollout status deploy/healthcheck-exporter --timeout=2m

PROM_SVC="prometheus-operated"
echo ">> Port-forward Prometheus: http://localhost:9090"
kubectl -n "${NAMESPACE}" port-forward svc/${PROM_SVC} 9090:9090 &
kubectl -n "${NAMESPACE}" port-forward svc/healthcheck-exporter 8080:8080 &