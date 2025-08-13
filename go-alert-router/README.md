# 🚨 Go Alert Router & Notifier

[![Go](https://img.shields.io/badge/Go-1.22+-blue.svg?style=flat-square&logo=go)](https://go.dev/)
[![Prometheus](https://img.shields.io/badge/metrics-Prometheus-orange?style=flat-square&logo=prometheus)](https://prometheus.io/)
[![Docker](https://img.shields.io/badge/Docker-ready-blue?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Serviço em **Golang** para **receber, deduplicar, agrupar e rotear alertas** para múltiplos canais, garantindo resiliência e observabilidade.

---

## 🚀 Funcionalidades
- **Recepção de alertas** via Webhook Prometheus, Loki, Sentry, etc.
- **Deduplicação inteligente** com fingerprint + janela de tempo.
- **Retries com backoff exponencial** + **Dead Letter Queue**.
- **Janela de silêncio (silences)** e **rate limit** por rota/canal.
- **Agrupamento de alertas (batch)** com templates (Go template).
- **Roteamento dinâmico** para:
  - Slack  
  - Microsoft Teams  
  - Email (SMTP)  
  - PagerDuty  
  - Webhooks genéricos
- **Calendário de plantão (on-call)** via iCal/CSV simples.
- **Persistência leve** (SQLite por padrão, Postgres opcional).
- **Métricas Prometheus** para ingestão, drop, retries e DLQ.
- **CLI + API REST** para gerenciamento de canais, regras e silences.
- **UI mínima (opcional)** via HTMX para administração rápida.

---

## 📂 Estrutura do Projeto
```text
go-alert-router/
├── cmd/                     # Entrypoints CLI/API
├── internal/                # Lógica principal
│   ├── router/               # Regras e roteamento
│   ├── notifier/             # Entregas e integrações
│   ├── storage/              # Persistência
│   └── scheduler/            # Agrupamento, retries, silences
├── configs/
│   ├── config.yaml           # Configuração principal
│   ├── oncall.csv            # Escala de plantão
│   └── grafana-dashboard.json# Dashboard para métricas
├── deployments/
│   ├── docker-compose.yaml
│   └── k8s/                  # Manifests para rodar no Kubernetes
├── dashboards/               # Dashboards Grafana
│   └── grafana-alert-router.json
├── Makefile                   # Comandos utilitários
└── README.md
```
##
### ⚙️ Configuração via config.yaml
```yaml
server:
  http_addr: ":8080"
  metrics_addr: ":9090"

storage:
  driver: sqlite
  dsn: data/alerts.db

routes:
  - match:
      severity: critical
    send_to: ["slack", "pagerduty"]
    rate_limit: "1m"
    group_window: "30s"

notifiers:
  slack:
    webhook_url: https://hooks.slack.com/services/...
  email:
    smtp_server: smtp.example.com
    smtp_user: alert@example.com
    smtp_pass: supersecret
    to: ops-team@example.com
```
##
### 📊 Métricas Prometheus
- `alert_router_ingested_total` — Total de alertas recebidos
- `alert_router_dropped_total` — Alertas descartados (silences, rate limit)
- `alert_router_delivered_total` — Entregas bem-sucedidas
- `alert_router_retry_total` — Retries realizados
- `alert_router_dlq_total` — Alertas na Dead Letter Queue
- `alert_router_delivery_latency_seconds` — Histograma de latência
##
### 🖥 Dashboard Grafana
- Dashboard incluído em `dashboards/grafana-alert-router.json` com:
- Total de alertas por status (ingested, dropped, delivered, DLQ)
- Retries por canal
- Latência p95/p99
- Distribuição por tipo de alerta
- Heatmap de alertas por hora
##
### 📦 Execução
Local
```bash
go mod tidy
go run cmd/api/main.go --config configs/config.yaml
```
Docker Compose
```bash
docker compose up --build
```
Kubernetes
```bash
kubectl apply -f deployments/k8s/
```
##
### 🧪 Testes
```bash
go test ./...
```
##
### 🔐 Segurança e RBAC (para K8s)
- ServiceAccount mínimo com permissão apenas para namespaces/recursos onde coleta alertas.
- Secrets no Kubernetes para armazenar tokens de Slack, SMTP e PagerDuty.
- TLS habilitado para API (opcional).
##
### 📜 Stack Docker Compose (API + Prometheus + Grafana)
Arquivos em `deployments/docker-compose.yaml` e `configs/.`
- Prometheus coleta métricas do Alert Router.
- Grafana com provisionamento automático do dashboard.
- Configuração pronta para rodar com:
```bash
make up
```
Acesse:
- API: http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
##
### 🛠 Makefile
```make
APP=alert-router

.PHONY: tidy build run docker up down logs fmt test

tidy:
	@go mod tidy

build:
	@mkdir -p bin
	@go build -o bin/$(APP) ./cmd/server

run:
	CONFIG_PATH=configs/config.yaml HTTP_ADDR=:8080 LOG_LEVEL=info ./bin/$(APP)

docker:
	docker build -t go-alert-router:local .

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

fmt:
	go fmt ./...

test:
	go test ./...
```
##
### 🔄 CI (GitHub Actions)
Arquivo `.github/workflows/ci.yml` incluído para build + testes automatizados:
```yaml
name: ci
on:
  push:
    branches: [ main ]
  pull_request:

jobs:
  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22.x"
      - name: Tidy
        run: go mod tidy
      - name: Build
        run: go build ./...
      - name: Test
        run: go test ./...
```
##
### 📌 Teste rápido do webhook
```bash
curl -XPOST http://localhost:8080/webhook/alertmanager \
  -H 'Content-Type: application/json' \
  -d '{
    "receiver": "default",
    "status": "firing",
    "alerts": [
      {
        "labels": {"alertname":"HighCPU","severity":"critical","instance":"web-1"},
        "annotations": {"summary":"CPU > 90%"},
        "startsAt": "2025-08-13T22:00:00Z"
      }
    ]
  }'
```
##
### 📄 Licença
MIT
