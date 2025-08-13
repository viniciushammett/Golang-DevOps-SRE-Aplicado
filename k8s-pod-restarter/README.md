# 🌀 Kubernetes Pod Restarter

**CLI + API + Scheduler** para reiniciar pods no Kubernetes com segurança e observabilidade.
- Reinício via **deleção de Pods** (controladores recriam).
- **API REST** para disparos on-demand.
- **Scheduler cron** a partir de YAML (operações rotineiras).
- **Métricas Prometheus** e endpoint `/healthz`.
- **RBAC** mínimo e manifests prontos.

## Sumário
- [Motivação](#motivação)
- [Modos de uso](#modos-de-uso)
- [Instalação](#instalação)
- [CLI](#cli)
- [API](#api)
- [Scheduler](#scheduler)
- [Métricas](#métricas)
- [Deploy no Kubernetes](#deploy-no-kubernetes)
- [RBAC](#rbac)
- [Boas práticas](#boas-práticas)
- [Licença](#licença)

## Motivação
Reiniciar pods é útil para:
- Renovar conexões/configs sem alterar Deployments.
- Mitigar pods zumbis com memória fragmentada.
- Rolar processos com leaks fora da janela de release.

## Modos de uso
- **CLI**: execução única (`restart`).
- **API**: `POST /restart` com payload JSON.
- **Scheduler**: cron via `configs/config.yaml`.

## Instalação
```bash
go mod tidy
go build -o bin/pod-restarter ./cmd/restarter
```
##
### CLI
```bash
pod-restarter restart \
  --selector "app=myapp,component=api" \
  --namespace prod \
  --max-age 30m \
  --grace-period 20s \
  --http-addr :8080
```
Flags principais:
- `--selector` **(obrigatório)** label selector (ex.: `app=myapp,component=api`)
- `--namespace` (vazio = todos)
- `--max-age` (ex.: `30m` → só reinicia pods mais antigos)
- `--grace-period` (padrão `30s`)
- `--dry-run`
- `--force` (ignora DaemonSets/StaticPods — **não recomendado**)
##
### API
Suba o servidor:
```bash
pod-restarter api --http-addr :8080 --api-token "SECRETO"
```
Chame:
```bash
curl -XPOST http://localhost:8080/restart \
  -H "Authorization: Bearer SECRETO" \
  -d '{"namespace":"prod","selector":"app=myapp,component=api","maxAge":"30m","gracePeriod":"20s"}' \
  -H "Content-Type: application/json"
```
##
### Scheduler
Arquivo `configs/config.yaml`
```yaml
jobs:
  - name: restart-api-nightly
    namespace: "prod"
    selector: "app=myapi,component=backend"
    schedule: "0 3 * * *"
    maxAge: "1h"
    gracePeriod: "30s"
```
Execute:
```bash
pod-restarter scheduler --config configs/config.yaml --http-addr :8080
```
##
### Métricas
Expostas em `/metrics:`
- `pod_restarter_restarts_total{namespace,reason}`
- `pod_restarter_errors_total{namespace,reason}`
- `pod_restarter_last_restart_timestamp{namespace}`
- `pod_restarter_operation_duration_seconds{namespace}`
##
### Deploy no Kubernetes
1. Namespace e RBAC:
```bash
kubectl create ns sre-tools
kubectl apply -f deploy/rbac.yaml
```
2. Deployment/Service/ServiceMonitor:
```bash
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/servicemonitor.yaml   # se usa Prometheus Operator
```
##
### RBAC
`ClusterRole` com permissões somente para `pods: get,list,watch,delete.`
##
### Boas práticas
- Comece com `--dry-run` em prod.
- Limite escopo via `--namespace` e seletores específicos.
- Evite `--force` (não reinicie DaemonSets/StaticPods).
- Acompanhe métricas e alerte para picos de reinício/erro.
##
### Licença
MIT License – veja o arquivo LICENSE para detalhes.
