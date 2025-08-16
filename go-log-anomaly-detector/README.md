# 🤖 Go Log Anomaly Detector

Pipeline de logs em Go com **detecção de anomalias via Machine Learning, tracing OpenTelemetry** e **frontend React** para visualização em tempo real.
Um projeto hands-on para aprender **observabilidade, MLOps e DevOps moderno.**

---

## ✨ Funcionalidades

- 🔄 Ingestão de logs em JSON/NDJSON.
- 📏 Regras customizadas (regex, thresholds, etc).
- 🚨 Alertas por Slack, Email ou Webhook.
- 📊 Métricas Prometheus expostas no /metrics.
- 🛰 Tracing distribuído via OpenTelemetry.
- 🧠 Detecção automática de anomalias com ML.
- 🌐 Frontend React para dashboards em tempo real.
- ⚙️ Deploy pronto para Docker Compose e Kubernetes.  

---

## 📂 Estrutura do Projeto

```plaintext
o-log-anomaly-detector/
├── cmd/                  # Entrypoints CLI/API
├── internal/
│   ├── ingest/           # Ingestão de logs
│   ├── rules/            # Regras customizadas
│   ├── alerts/           # Notificações
│   ├── storage/          # Persistência BoltDB
│   ├── metrics/          # Exposição Prometheus
│   └── tracing/          # OpenTelemetry
├── ml/
│   ├── trainer.py        # Treino do modelo ML
│   ├── detector.go       # Inferência Go
│   └── model.stats.json  # Modelo salvo
├── frontend/             # WebApp React
│   ├── src/
│   └── package.json
├── configs/
│   ├── config.yaml
│   └── grafana-dashboard.json
├── deployments/
│   ├── docker-compose.yaml
└── README.md
```
##
### 🚀 Como rodar
**Backend Go**
```bash
go run ./cmd/server/main.go --config ./configs/config.yaml
```
**Frontend React**
```bash
cd frontend
npm install
npm run dev

```
##
### 📊 Observabilidade (OpenTelemetry)
🔎 **Tracing com OpenTelemetry**
- Ativado em todas as etapas de ingestão e regras.
- Exportador padrão: OTLP gRPC.
- Pode ser integrado com Jaeger, Tempo ou Grafana Cloud.
##
### 📈 Métricas Prometheus
- Expostas no endpoint: http://localhost:8080/metrics
- Dashboard pronto em configs/grafana-dashboard.json.
##
### 🧠 Machine Learning
- `trainer.py` → treina modelos em batch (Isolation Forest).
- Input: `ml/logs.csv` (extraído da base BoltDB).
- Output: `ml/model.stats.json` com thresholds e pesos.
- Inferência em tempo real feita pelo `detector.go`.

Treinar Manualmente
```bash
python3 ml/trainer.py --input ml/logs.csv --output ml/model.stats.json
```
##
### 🌐 Frontend Web (React)
- Visualização de logs em tempo real (via SSE/WS).
- Lista de anomalias detectadas.
- Dashboard com métricas (latência, taxa de erro, outliers).
- Construído com Vite + React + Tailwind + Recharts.

Rodar localmente:
```bash
cd frontend
npm install
npm run dev
```
##
### 📤 Exportador de Logs
O utilitário `tools/export_logs_csv.go` gera CSV para treino:
```bash
make export-logs
# gera ml/logs.csv
```
Estrutura do CSV:
```bash
ts,source,msg
2025-08-15T10:00:00Z,app-1,"erro crítico: conexão perdida"
```
##
### ⚙️ CI/CD e Automação
O projeto vem com pipelines GitHub Actions prontos para build, testes e MLOps.

### 🔄 Treino Automático + Manual (`ml-train.yml`)
- Executa diariamente às **03h (BRT)** ou sob demanda.
- Exporta logs → `ml/logs.csv.`
- Roda o `trainer.py` e gera `ml/model.stats.json.`
- Publica artefato e abre **PR automático** com o modelo.
##
### ✅ Validação de Dataset em PRs (`ml-validate-dataset.yml`)
- Dispara em mudanças no `ml/logs.csv` ou `ml/trainer.py.`
- Roda `trainer.py` para validar consistência.
- Gera modelo temporário como artefato.
- Evita merges com datasets inválidos.
##
### 📌 Resumo:
- `ml-train.yml` mantém o modelo sempre atualizado.
- `ml-validate-dataset.yml` protege a qualidade do dataset.
- Ambos garantem confiabilidade no pipeline de ML.
##
### 📦 Deploy
**Docker Compose**
```bash
docker-compose up -d
```
##
### 📜 Licença

MIT © 2025
