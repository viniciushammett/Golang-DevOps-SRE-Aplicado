# go-sre-monitor

Monitor HTTP minimalista em **Go** com **métricas Prometheus**, **healthcheck**, **logs estruturados** e **config YAML**. Ideal para labs de **DevOps/SRE**, demos de observabilidade e checks de disponibilidade simples.

## Recursos
- Checagens paralelas de URLs (GET/HEAD) com timeout por serviço
- Métricas: `probe_up`, `probe_duration_seconds`, `probe_failures_total`, `slo_latency_breaches_total`
- Endpoints: `/metrics`, `/healthz`
- Logs em JSON (zerolog), níveis por env
- Docker + Docker Compose com Prometheus e Grafana (dashboard pronto)

## Início rápido
```bash
make tidy
make run
# http://localhost:8080/metrics
