# 🔍 Go Log Anomaly Detector

Serviço em **Golang** para **detecção de anomalias em logs em tempo real**, aplicando regras configuráveis em YAML.  
Recebe logs via API, Kafka ou leitura de arquivos locais, e emite alertas para múltiplos canais (Slack, Discord, Webhook, Email).  

Inclui suporte a métricas Prometheus, dashboards Grafana, auditoria em banco de dados e execução containerizada.  

---

## 🚀 Funcionalidades

- 📥 **Ingestão de logs** via:
  - HTTP API (`/ingest`)
  - Streams Kafka
  - Arquivos locais (tailing)
- 📜 **Regras configuráveis em YAML**:
  - Expressões regulares
  - Threshold de ocorrências em janelas de tempo
  - Severidade (info, warn, critical)
- ⚠️ **Alertas automáticos** para:
  - Slack
  - Discord
  - Webhook genérico
  - Email (SMTP)
- 📊 **Observabilidade nativa**:
  - Métricas em `/metrics` (Prometheus)
  - Healthcheck em `/healthz`
  - Dashboard Grafana pronto
- 🗄️ **Persistência**:
  - SQLite para auditoria local
  - PostgreSQL opcional para produção
- 🐳 **Deploy facilitado**:
  - Dockerfile
  - docker-compose
  - Manifests Kubernetes

---

## 📂 Estrutura do Projeto

```bash
go-log-anomaly-detector/
├── cmd/                   # Entrypoints (API/worker)
│   └── main.go
├── internal/              # Lógica principal
│   ├── ingest/            # Consumo de logs (API, Kafka, File)
│   ├── rules/             # Engine de regras YAML
│   ├── alerts/            # Slack, Discord, Email, Webhook
│   ├── storage/           # Persistência (SQLite/Postgres)
│   └── metrics/           # Exposição Prometheus/Healthz
├── configs/
│   ├── config.yaml        # Configuração geral
│   └── rules.yaml         # Regras de detecção
├── deployments/
│   ├── docker-compose.yaml
│   
├── dashboards/
│   └── grafana.json       # Dashboard de métricas/anomalias
└── README.md
```
##
### ⚙️ Exemplo de Configuração
`config.yaml`
```yaml
server:
  port: 8080

ingestion:
  kafka:
    brokers: ["localhost:9092"]
    topic: "logs"
  file:
    path: "/var/log/app.log"

alerts:
  slack:
    webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
  email:
    smtp: "smtp.gmail.com:587"
    user: "alerts@company.com"
    pass: "supersecret"
    to: ["devops@company.com"]
```
`rules.yaml`
```yaml
rules:
  - name: "Erro crítico no sistema"
    regex: "CRITICAL|FATAL"
    threshold: 1
    window: "1m"
    severity: "critical"

  - name: "Múltiplas falhas de login"
    regex: "login failed"
    threshold: 5
    window: "10m"
    severity: "warn"
```
##
### ▶️ Execução
**Via Go**
```bash
go run cmd/main.go --config configs/config.yaml
```
**Via Docker**
```bash
docker build -t go-log-anomaly-detector .
docker run -p 8080:8080 go-log-anomaly-detector
```
**Via Docker Compose**
```bash
docker-compose up -d
```
##
### 📊 Observabilidade
- **Métricas Prometheus →** http://localhost:8080/metrics
- **Healthcheck →** http://localhost:8080/healthz
- **Dashboard Grafana →** importe `dashboards/grafana.json`
##
### 🔄 Fluxo de Trabalho
1) Logs são recebidos via API/Kafka/File.
2) Engine de regras aplica regex + thresholds em janelas de tempo.
3) Quando regra é acionada → alerta é disparado.
4) Evento é persistido no banco (SQLite/Postgres).
5) Métricas são expostas para Prometheus/Grafana.
##
### 🧪 CI/CD
- **Lint & Testes:** GolangCI-Lint + Go test
- **Build:** Docker + multi-stage build
- **Policy QA:** validação YAML de regras
- **Release:** tags automáticas com GitHub Actions
