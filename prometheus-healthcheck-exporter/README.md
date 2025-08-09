# Prometheus Healthcheck Exporter (Go)

Mini-exporter Prometheus em Go que verifica URLs HTTP periodicamente e expõe métricas de **UP/DOWN**, **latência (ms)** e **status code** em `/metrics`. Ideal para portfolio DevOps/SRE e para aprender Go com um caso real.

## ✨ Recursos
- Checagem de múltiplas URLs com **concorrência configurável**
- Intervalo e timeout configuráveis
- Exposição de métricas no formato Prometheus (`/metrics`)
- Binário estático e imagem Docker **não-root (distroless)**
- Código simples e idiomático em Go

## 🚀 Executar localmente

### Como inicializar o projeto

```bash
mkdir -p prometheus-healthcheck-exporter
cd prometheus-healthcheck-exporter
go mod init github.com/seuusuario/prometheus-healthcheck-exporter
# crie o main.go com o conteúdo acima
go run .
```

```bash
go run . \
  -urls "https://example.com,https://httpbin.org/status/204,https://httpbin.org/status/500" \
  -interval 30s -timeout 3s -concurrency 5 -port :8080
```

```bash
# Ver métricas
curl http://localhost:8080/metrics
```

🔧 Flags
Flag	Default	Descrição
-urls	https://example.com,https://httpbin.org/status/204,https://httpbin.org/status/500	Lista de URLs separadas por vírgula
-interval	30s	Intervalo de checagem (ex.: 15s, 1m)
-timeout	3s	Timeout por requisição
-port	:8080	Host/porta do servidor HTTP (:8080, 0.0.0.0:8080 etc.)
-concurrency	5	Máximo de checagens simultâneas por rodada

Endpoints:

GET /metrics — métricas para Prometheus

GET /healthz — liveness do exporter


🐳 Docker
Build:

```bash
make docker-build
```
Run:

bash
Copiar
Editar
make docker-run PORT=8080 URLS="https://example.com,https://httpbin.org/status/500" INTERVAL=15s TIMEOUT=2s CONCURRENCY=5
# depois:
curl http://localhost:8080/metrics
📈 Prometheus (scrape config)
yaml
Copiar
Editar
scrape_configs:
  - job_name: 'healthcheck_exporter'
    static_configs:
      - targets: ['localhost:8080']
☸️ Kubernetes (exemplo rápido)
yaml
Copiar
Editar
apiVersion: apps/v1
kind: Deployment
metadata:
  name: healthcheck-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: healthcheck-exporter
  template:
    metadata:
      labels:
        app: healthcheck-exporter
    spec:
      containers:
        - name: exporter
          image: healthcheck-exporter:latest
          args:
            - -urls=https://example.com,https://httpbin.org/status/500
            - -interval=30s
            - -timeout=3s
            - -concurrency=5
            - -port=:8080
          ports:
            - containerPort: 8080
          securityContext:
            runAsNonRoot: true
            allowPrivilegeEscalation: false
---
apiVersion: v1
kind: Service
metadata:
  name: healthcheck-exporter
spec:
  selector:
    app: healthcheck-exporter
  ports:
    - port: 8080
      targetPort: 8080
      name: http
Prometheus Operator (opcional - ServiceMonitor):

yaml
Copiar
Editar
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: healthcheck-exporter
spec:
  selector:
    matchLabels:
      app: healthcheck-exporter
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
🧪 Testes
Estrutura pronta para adicionar testes:

bash
Copiar
Editar
make test
🔒 Segurança
Imagem distroless e não-root

Somente porta HTTP exposta

Sem persistência de dados

🗺️ Roadmap
Retries/backoff por URL (com métricas de tentativa)

Labels extras (hostname, grupo, ambiente)

Config via arquivo YAML/ENV

Histogram para tempo em vez de gauge

Healthcheck TCP/HTTPS avançado (SNI, headers custom)