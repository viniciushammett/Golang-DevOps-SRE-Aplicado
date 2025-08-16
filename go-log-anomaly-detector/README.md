# 🔍 Go Log Anomaly Detector

Detecção de **anomalias em logs** em tempo real (Go). Regras YAML (regex + janelas), alertas Slack/Webhook, métricas Prometheus e dashboard Grafana.

## Recursos
- Ingestão HTTP (`POST /v1/logs`) — stubs para Kafka/arquivo
- Regras em `configs/rules.example.yaml` (regex, janela, limiares)
- Armazenamento de anomalias (BoltDB) e consulta `GET /v1/anomalies`
- Métricas Prometheus (`/metrics`) + dashboard Grafana
- Alertas Slack (webhook)

## Estrutura
```text
cmd/server, internal/{detector,rules,ingest,store,metrics,notify}
configs/{config.yaml,rules.example.yaml}, dashboards/, examples/
```