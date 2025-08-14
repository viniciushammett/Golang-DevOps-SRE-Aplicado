# 🛡 Go Access Auditor

Auditoria centralizada de **acessos e comandos** (Linux/Kubernetes/DBs). Inclui **Agente**, **API/Coletor**, **Dashboard HTMX**, **Métricas Prometheus**, **Export CSV** e **alertas Slack** para comandos sensíveis.

## Recursos
- Agente envia eventos (stdin/linha de comando): `user@host`, `source`, `command`
- API: `POST /v1/events`, `GET /v1/events`, `GET /v1/export.csv`, `GET /metrics`, `GET /`
- Regras sensíveis via **regex** (config)
- Notificação **Slack** opcional
- Dashboard simples embutido (HTMX)
- Docker/Compose + Grafana dashboard

## Estrutura
```text
go-access-auditor/
├── cmd/ (server, agent)
├── internal/ (api, store, rules, ingest, metrics, notify)
├── web/static/index.html
├── configs/config.yaml
├── dashboards/grafana-access-auditor.json
└── README.md
```
