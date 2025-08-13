# 📚 Go Log Aggregator

Agregador de logs em **Go** com **tail em tempo real**, **API de consulta** e **métricas Prometheus**. Ideal para laboratórios DevOps/SRE, troubleshooting e POCs de observabilidade.

## Recursos
- Ingestão de múltiplas fontes:
  - **Arquivos** (tail não intrusivo com detecção de rotação simples)
  - **HTTP Pull** (faz GET periódico e agrega respostas)
  - **stdin** (opcional)
- **Buffer circular** em memória (ring) configurável (default 10k linhas).
- **API HTTP** `/logs` com filtros:
  - `since` (segundos ou RFC3339), `until`
  - `include`/`exclude` (regex)
  - `source`, `limit`, `offset`
- **/metrics** Prometheus e **/healthz**.
- Docker/Compose + Prometheus/Grafana (opcionais).

## Início rápido
```bash
make tidy
make build
./bin/logagg
# ou com Docker:
docker compose up --build
```
##
### Configuração (`configs/config.yaml`)
```yaml
bufferSize: 10000
sources:
  stdin: { enabled: false }
  files:
    - name: "syslog"
      path: "/var/log/syslog"
      pollInterval: 300ms
  http:
    - name: "status-endpoint"
      url: "http://localhost:9000/status"
      interval: 5s
```
Endpoints
- `GET /healthz` → ok
- `GET /metrics` → métricas Prometheus
- `GET /logs` → consulta de logs
  - Ex.: `GET /logs?source=syslog&include=ERROR&since=600&limit=100`
Exemplos
```bash
# últimos 5 minutos contendo "ERROR"
curl 'http://localhost:8080/logs?include=ERROR&since=300&limit=50'

# apenas da fonte "app", ignorando linhas com "DEBUG"
curl 'http://localhost:8080/logs?source=app&exclude=DEBUG&limit=200'

# janela de tempo absoluta
curl 'http://localhost:8080/logs?since=2025-08-13T21:00:00Z&until=2025-08-13T21:30:00Z'
```
Métricas
- `logagg_logs_ingested_total{source}`
- `logagg_ingest_errors_total{source}`
- `logagg_ingest_latency_seconds{source}`
- `logagg_api_queries_total{route}`

Boas práticas
- Ajuste `bufferSize` conforme tráfego.
- Use `include/exclude` para reduzir payloads.
- Monte volumes de logs como **read-only** no container.
- Para fontes HTTP, padronize a resposta (texto simples por linha).
##
### 🧪 Makefile (opcional)
```make
APP=logagg

.PHONY: build run tidy test docker
build:
	@mkdir -p bin
	go build -o bin/$(APP) ./cmd/aggregator

run:
	CONFIG_PATH=configs/config.yaml HTTP_ADDR=:8080 LOG_LEVEL=info go run ./cmd/aggregator

tidy:
	go mod tidy

docker:
	docker build -t go-log-aggregator:local .
