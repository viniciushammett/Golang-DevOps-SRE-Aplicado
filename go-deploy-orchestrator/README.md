# 🚀 Go Deploy Orchestrator

Orquestrador de deploys e **rollback automático** em **Go**, com estratégias **canary** e **blue-green**, histórico, aprovação manual e **métricas Prometheus**.

[![Go](https://img.shields.io/badge/Go-1.22+-blue.svg?style=flat-square&logo=go)](https://go.dev/)
[![Prometheus](https://img.shields.io/badge/Metrics-Prometheus-orange?style=flat-square&logo=prometheus)](https://prometheus.io/)
[![Docker](https://img.shields.io/badge/Docker-ready-blue?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## ✨ Recursos
- **API + CLI** para iniciar deploys
- Estratégias: **Canary** (steps) e **Blue-Green**
- **Rollback automático** quando SLOs são violados (Prometheus)
- **Aprovação manual** opcional (`/deploys/{id}/approve`)
- **Histórico em BoltDB**, métricas Prometheus e dashboard Grafana
- Manifests K8s (RBAC, Deployment, Service, ServiceMonitor)

---

## 📂 Estrutura do Projeto
```text
go-deploy-orchestrator/
├── cmd/                     # Entrypoints (server + CLI doctl)
│   ├── server/
│   └── doctl/
├── internal/                # Lógica principal
│   ├── api/                 # HTTP server (endpoints)
│   ├── config/              # Leitura de configs
│   ├── k8s/                 # Cliente e operações em Deployments
│   ├── logger/              # Zerolog wrapper
│   ├── metrics/             # Prometheus collectors
│   ├── orchestrator/        # Orquestração de deploys
│   ├── prometheus/          # Avaliador de consultas
│   ├── strategies/          # canary.go / bluegreen.go
│   ├── store/               # BoltDB (histórico)
│   └── util/                # retry/backoff etc
├── configs/
│   └── config.yaml          # Config principal (PromQL/thresholds/defauts)
├── examples/
│   └── myapp/               # Manifests do app de exemplo
│      ├── deployment.yaml
│      └── service.yaml
├── deploy/                  # Manifests do orquestrator para K8s
│   ├── rbac.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── servicemonitor.yaml
├── dashboards/
│   └── grafana-deploy-orchestrator.json
├── .github/workflows/
│   ├── deploy-canary.yml
│   └── deploy-bluegreen.yml
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```
##
### ⚙️ Configuração (`configs/config.yaml`)
```yaml
server:
  httpAddr: ":8080"

kube:
  kubeconfig: ""     # vazio = in-cluster; senão, caminho do kubeconfig local
  context: ""

prometheus:
  url: "http://prometheus:9090"
  timeout: 10s
  queries:
    errorRate: 'sum(rate(http_requests_total{job="myapp",status=~"5.."}[5m])) / sum(rate(http_requests_total{job="myapp"}[5m]))'
    p95: 'histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job="myapp"}[5m])) by (le))'
  thresholds:
    maxError: 0.02   # 2% erro
    maxP95: 0.5      # 500ms
  window: "5m"

storage:
  path: "data/deploy-orchestrator.db"

defaults:
  canaryStepPercent: 20
  canaryPauseSec: 45

authToken: ""        # opcional: define para proteger /deploys/*/approve
```
##
### ▶️ Executando
Local
```bash
go mod tidy
go build -o bin/orchestrator ./cmd/server
CONFIG_PATH=configs/config.yaml ./bin/orchestrator
```
Docker Compose
```bash
docker compose up --build
```
Kubernetes
```bash
kubectl create ns sre-tools
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/servicemonitor.yaml   # se usar Prometheus Operator
```
##
### 🌐 Endpoints
- `POST /deploys` → inicia deploy (canary/bluegreen)
- `GET /deploys` → lista histórico
- `GET /deploys/{id}` → status
- `POST /deploys/{id}/approve` → libera quando requireApproval=true
- `GET /metrics, GET /healthz`

Exemplo (curl)
```bash
curl -XPOST :8080/deploys -H 'Content-Type: application/json' -d '{
  "app":"myapp","namespace":"default","image":"repo/myapp:1.2.3",
  "strategy":"canary",
  "params":{"canaryStep":"25","canaryPause":"45","maxError":"0.02","maxP95":"0.5"},
  "requireApproval": false
}'
```
##
### 📊 Dashboard Grafana
Importe `dashboards/grafana-deploy-orchestrator.json` e monitore:
- `do_deploys_started_total{app,strategy}`
- `do_deploys_succeeded_total{app,strategy}`
- `do_deploys_failed_total{app,strategy,reason}`
- `do_step_duration_seconds_bucket{app,strategy,step}`
##
### 🧪 Exemplo prático — Canary do myapp (5 réplicas, step 20%)
1) Suba o app base
```bash
kubectl apply -f examples/myapp/deployment.yaml
kubectl apply -f examples/myapp/service.yaml
```
2) Ajuste Prometheus/thresholds
Edite `configs/config.yaml` (queries e limites), por exemplo:
```yaml
dprometheus:
  url: "http://prometheus:9090"
  timeout: 10s
  thresholds: { maxError: 0.02, maxP95: 0.5 }
  window: "5m"
defaults: { canaryStepPercent: 20, canaryPauseSec: 45 }
```
3) Dispare o canary
```bash
curl -XPOST http://ORCHESTRATOR_HOST:8080/deploys \
  -H 'Content-Type: application/json' \
  -d '{
    "app": "myapp",
    "namespace": "default",
    "image": "repo/myapp:1.2.3",
    "strategy": "canary",
    "params": { "canaryStep": "20", "canaryPause": "45", "maxError": "0.02", "maxP95": "0.5" },
    "requireApproval": false
  }'
```
4) Acompanhe
```bash
curl http://ORCHESTRATOR_HOST:8080/deploys | jq .
```
##
### 🟦🟩 Exemplo prático — Blue-Green do myapp (rollout completo + SLO)
1) Base (se necessário)
```bash
kubectl apply -f examples/myapp/deployment.yaml
kubectl apply -f examples/myapp/service.yaml
```
2) Dispare Blue-Green
```bash
curl -XPOST http://ORCHESTRATOR_HOST:8080/deploys \
  -H 'Content-Type: application/json' \
  -d '{
    "app": "myapp",
    "namespace": "default",
    "image": "repo/myapp:1.2.3",
    "strategy": "bluegreen",
    "params": { "probeWait": "30", "maxError": "0.02", "maxP95": "0.5" },
    "requireApproval": false
  }'
```
3) Acompanhe
```bash
curl http://ORCHESTRATOR_HOST:8080/deploys | jq .
```
- **Como funciona nesta estratégia**: troca a imagem no mesmo `Deployment`, aguarda o rollout, espera `probeWait` e consulta os SLOs no Prometheus. Se violar, **rollback** automático para a imagem anterior.
- **Opcional (referência)**: para “Blue-Green clássico” com dois Deployments (`track: blue|green`) e comutação de `Service`, adapte seus manifests conforme sua topologia.
##
### 🧰 Makefile (alvos úteis)
- Configure variáveis no `.env` (opcional):
```dotenv
APP_NAME=myapp
APP_NAMESPACE=default
IMAGE=repo/myapp:1.2.3
ORCHESTRATOR_URL=http://localhost:8080
ORCHESTRATOR_TOKEN=
CANARY_STEP=20
CANARY_PAUSE=45
MAX_ERROR=0.02
MAX_P95=0.5
PROBE_WAIT=30
```
Targets
```make
# ===========================
# App de exemplo (myapp)
# ===========================
myapp-apply:
	kubectl apply -f examples/myapp/deployment.yaml
	kubectl apply -f examples/myapp/service.yaml

myapp-delete:
	-kubectl delete -f examples/myapp/service.yaml
	-kubectl delete -f examples/myapp/deployment.yaml

# ===========================
# Canary
# ===========================
canary-start:
	@echo "🚀 Canary para $(APP_NAME) -> $(IMAGE)"
	curl -sS -XPOST "$(ORCHESTRATOR_URL)/deploys" \
	  -H 'Content-Type: application/json' \
	  -H "Authorization: Bearer $(ORCHESTRATOR_TOKEN)" \
	  -d '{
	    "app":"$(APP_NAME)",
	    "namespace":"$(APP_NAMESPACE)",
	    "image":"$(IMAGE)",
	    "strategy":"canary",
	    "params":{"canaryStep":"$(CANARY_STEP)","canaryPause":"$(CANARY_PAUSE)","maxError":"$(MAX_ERROR)","maxP95":"$(MAX_P95)"},
	    "requireApproval": false
	  }' | jq .

canary-status:
	curl -sS "$(ORCHESTRATOR_URL)/deploys" -H "Authorization: Bearer $(ORCHESTRATOR_TOKEN)" | jq .

# ===========================
# Blue-Green
# ===========================
bluegreen-start:
	@echo "🟦🟩 Blue-Green para $(APP_NAME) -> $(IMAGE)"
	curl -sS -XPOST "$(ORCHESTRATOR_URL)/deploys" \
	  -H 'Content-Type: application/json' \
	  -H "Authorization: Bearer $(ORCHESTRATOR_TOKEN)" \
	  -d '{
	    "app": "$(APP_NAME)",
	    "namespace": "$(APP_NAMESPACE)",
	    "image": "$(IMAGE)",
	    "strategy": "bluegreen",
	    "params": { "probeWait": "$(PROBE_WAIT)", "maxError": "$(MAX_ERROR)", "maxP95": "$(MAX_P95)" },
	    "requireApproval": false
	  }' | jq .

bluegreen-status:
	curl -sS "$(ORCHESTRATOR_URL)/deploys" -H "Authorization: Bearer $(ORCHESTRATOR_TOKEN)" | jq .
```
##
### 🤖 CI (GitHub Actions)
- **Canary:** `.github/workflows/deploy-canary.yml`
  - Build/push da imagem (Buildx) e **disparo do canary** com polling até `succeeded/rolled_back.`
- **Blue-Green:** `.github/workflows/deploy-bluegreen.yml`
  - Build/push e **disparo blue-green** com **polling.**

**Secrets:**
- `ORCHESTRATOR_URL` → URL base do orquestrator (ex.: `http://orchestrator.svc.cluster.local:8080`)
- `ORCHESTRATOR_TOKEN` → opcional, se authToken estiver ativo
- (Se usar GHCR) `GITHUB_TOKEN` já possui `packages:write.`
##
### 🧭 Boas práticas
- Em produção, comece com `requireApproval=true.`
- Ajuste `canaryStep/canaryPause` de acordo com tráfego real.
- Garanta que suas queries PromQL **representem o SLO real** do serviço.
- Proteja `/approve` com `authToken.`
- Acompanhe métricas e dashboard durante os rollouts.
##
### 🛠 Troubleshooting
- **Rollout não finaliza:** verifique `readinessProbe` e eventos do Deployment.
- **SLOs sempre 0:** ajuste rótulos/queries no Prometheus; teste no console do Prometheus.
- **Rollback não voltou imagem antiga:** confira permissões RBAC e snapshot de `ImageOld`.
- **CI timeout:** aumente tentativas no polling (workflow) ou `canaryPause`.
##
### 📄 Licença

MIT
