<div align="center">
  <h1>📜 Logwatcher</h1>
  <p>Monitoramento de logs com alertas e métricas para Prometheus — desenvolvido em Go</p>
</div>

---

## 📖 Descrição

O **Logwatcher** é uma ferramenta em Go para monitorar arquivos de log, detectar padrões de interesse (como erros críticos) e:

- Enviar alertas para **Slack/Discord** via Webhook.
- Expor métricas no formato **Prometheus** (`/metrics`).
- Trabalhar com múltiplos arquivos em paralelo, incluindo rotação de logs.
- Evitar spam através de **cooldown** e agrupamento de eventos.

Este projeto foi criado como parte da trilha de aprendizado prático de **DevOps/SRE com Golang**.

---

## ✨ Recursos Principais

- **Múltiplos arquivos** com `filepath.Glob` (`-files "/var/log/nginx/*.log"`).
- **Detecção de rotação** de logs por inode/tamanho.
- **Buffer de deduplicação** e janela de cooldown.
- **Webhook** Slack/Discord configurável via flags/env.
- **/metrics** Prometheus com contadores por padrão e arquivo.

---

## 🚀 Como Usar

### Instalação local

```bash
git clone https://github.com/<seu-usuario>/<seu-repo>.git
cd logwatcher

go mod init github.com/<seu-usuario>/logwatcher
go mod tidy

go build -o logwatcher .
```
## Execução básica

```bash
./logwatcher \
  -files="/var/log/syslog" \
  -pattern="(?i)(error|critical)" \
  -metrics-addr=":9100"
```
## Com Webhook para Slack

```bash
./logwatcher \
  -files="/var/log/syslog" \
  -pattern="(?i)(error|critical)" \
  -metrics-addr=":9100" \
  -webhook="$WEBHOOK_URL" \
  -channel="alerts" \
  -title="[prod]"
```
## 🐳 Docker

Build
```bash
docker build -t logwatcher:latest .
```
Execução
```bash
docker run --rm \
  -v /var/log:/var/log:ro \
  -p 9100:9100 \
  logwatcher:latest \
  -files=/var/log/syslog \
  -pattern="(?i)(error|critical)" \
  -metrics-addr=:9100
```
## ☸️ Kubernetes

O Logwatcher pode ser executado no Kubernetes monitorando logs do host ou de aplicações específicas.

Namespace
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: logwatcher
```
Deployment + Service
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logwatcher
  namespace: logwatcher
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logwatcher
  template:
    metadata:
      labels:
        app: logwatcher
    spec:
      volumes:
        - name: varlog
          hostPath:
            path: /var/log/containers
            type: Directory
      containers:
        - name: logwatcher
          image: logwatcher:latest
          args:
            - -files=/hostlogs/*.log
            - -pattern=(?i)(error|critical|panic)
            - -metrics-addr=:9100
          ports:
            - name: http-metrics
              containerPort: 9100
          volumeMounts:
            - name: varlog
              mountPath: /hostlogs
              readOnly: true
          securityContext:
            runAsNonRoot: true
            runAsUser: 65532
```
Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: logwatcher
  namespace: logwatcher
spec:
  selector:
    app: logwatcher
  ports:
    - name: http-metrics
      port: 9100
      targetPort: http-metrics
```
## 📊 Integração com Prometheus
Se você usa Prometheus Operator, crie um `ServiceMonitor`:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: logwatcher
  namespace: logwatcher
spec:
  selector:
    matchLabels:
      app: logwatcher
  endpoints:
    - port: http-metrics
      path: /metrics
      interval: 30s
```
---
## 📜 Flags disponíveis

| Flag             | Descrição                               |
| ---------------- | --------------------------------------- |
| `-files`         | Arquivos de log (suporta glob)          |
| `-pattern`       | Expressão regular para detecção         |
| `-metrics-addr`  | Endereço para expor métricas Prometheus |
| `-webhook`       | URL do Webhook Slack/Discord            |
| `-channel`       | Canal/destino do alerta                 |
| `-title`         | Título prefixo da notificação           |
| `-poll`          | Intervalo de leitura dos logs           |
| `-cooldown`      | Janela mínima entre alertas             |
| `-bundle-window` | Janela para agrupar eventos             |
| `-bundle-max`    | Máximo de eventos agrupados             |

---

## 🐞 Troubleshooting
- Logs não encontrados: verifique o caminho usado em `-files`.
- Webhook não envia: valide a URL e permissões no destino.
- Sem métricas no Prometheus: confirme se `/metrics` está exposto e o ServiceMonitor configurado.

---

📄 Licença
Este projeto está sob a licença MIT. Veja o arquivo LICENSE para mais detalhes.
