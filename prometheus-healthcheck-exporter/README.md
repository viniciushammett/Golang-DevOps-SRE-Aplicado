## ✨ Visão Geral
O **Prometheus Healthcheck Exporter** é um mini-exporter escrito em **Golang** para verificar periodicamente URLs HTTP/HTTPS e expor métricas no formato **Prometheus**.

🎯 Ideal para:
- Times **DevOps/SRE** que precisam monitorar endpoints críticos  
- Labs de **observabilidade**  
- Demonstração de skills em **Go** + **Prometheus** + **Kubernetes**  

---

## ✨ Recursos
- Checagem de múltiplas URLs com **concorrência configurável**
- Intervalo e timeout configuráveis
- Exposição de métricas no formato Prometheus (`/metrics`)
- Binário estático e imagem Docker **não-root (distroless)**
- Código simples e idiomático em Go

## 🚀 Executar localmente

```bash
go run . \
  -urls "https://example.com,https://httpbin.org/status/204" \
  -interval 30s -timeout 3s -concurrency 5 -port :8080

curl http://localhost:8080/metrics

```

| Flag           | Default                                                                             | Descrição                                                  |
| -------------- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `-urls`        | `https://example.com,https://httpbin.org/status/204,https://httpbin.org/status/500` | Lista de URLs separadas por vírgula                        |
| `-interval`    | `30s`                                                                               | Intervalo de checagem (ex.: `15s`, `1m`)                   |
| `-timeout`     | `3s`                                                                                | Timeout por requisição                                     |
| `-port`        | `:8080`                                                                             | Host/porta do servidor HTTP (`:8080`, `0.0.0.0:8080` etc.) |
| `-concurrency` | `5`                                                                                 | Máximo de checagens simultâneas por rodada                 |

🐳 Docker
Build:

```bash
docker build -t healthcheck-exporter:latest .
docker run --rm -p 8080:8080 healthcheck-exporter:latest \
  -urls=https://example.com -interval=15s -timeout=2s
```
Run:

```bash
make docker-run PORT=8080 URLS="https://example.com,https://httpbin.org/status/500" INTERVAL=15s TIMEOUT=2s CONCURRENCY=5
# depois:
curl http://localhost:8080/metrics
```

☸️ Kubernetes
Stack completa com Prometheus Operator + kind: 📂 k8s-exporter-stack.yaml
```bash
kind create cluster --name monitoring-lab --config kind-cluster.yaml
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install lab-prom prometheus-community/kube-prometheus-stack -n monitoring-lab --create-namespace
kubectl apply -f k8s-exporter-stack.yaml

```
Acesse:
```bash
Exporter: kubectl port-forward svc/healthcheck-exporter 8080:8080 -n monitoring-lab
Prometheus: kubectl port-forward svc/prometheus-operated 9090:9090 -n monitoring-lab
```

🧪 Testes
Estrutura pronta para adicionar testes:

```bash
make test
```

🔒 Segurança
- Imagem distroless e não-root
- Somente porta HTTP exposta
- Sem persistência de dados

🗺️ Roadmap
- Retries/backoff por URL (com métricas de tentativa)
- Labels extras (hostname, grupo, ambiente)
- Config via arquivo YAML/ENV
- Histogram para tempo em vez de gauge
- Healthcheck TCP/HTTPS avançado (SNI, headers custom)
