<div align="center">
  <h1>🧾 logwatcher</h1>
  <p>Tail de logs em Go com regex, rotação, múltiplos arquivos, deduplicação/cooldown, métricas Prometheus e alertas via webhook</p>

  <img src="https://img.shields.io/badge/Go-1.22+-blue?style=flat-square&logo=go" />
  <img src="https://img.shields.io/badge/platform-linux%20%7C%20macos-lightgrey?style=flat-square" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" />
</div>

---

## 📖 Descrição

O **logwatcher** segue (tail -f) um ou vários arquivos de log, aplica um filtro por **regex**, trata **rotação/truncamento** automaticamente, agrupa ocorrências com **deduplicação/cooldown** para evitar spam, expõe **métricas Prometheus** em `/metrics` e envia **alertas via webhook** (Slack/Discord).

---

## ✨ Recursos

- **Regex**: filtre linhas em tempo real (ex.: `(?i)error|critical`).
- **Rotação**: detecta rename/remove/create e truncamento com `fsnotify`.
- **Múltiplos arquivos**: use `-files` com glob (ex.: `"/var/log/nginx/*.log,/var/log/app/*.log"`).
- **Deduplicação/Cooldown**: agrupa N ocorrências em X segundos antes de alertar.
- **Prometheus**: expõe métricas úteis de leitura e matches.
- **Webhooks (Slack/Discord)**: envia payload consolidado com título/canal.

---

## 🛠 Instalação

```bash
git clone https://github.com/viniciushammett/Golang-DevOps-SRE-Aplicado.git
cd Golang-DevOps-SRE-Aplicado/logwatcher

go mod init github.com/viniciushammett/Golang-DevOps-SRE-Aplicado/logwatcher
go get github.com/fsnotify/fsnotify
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go mod tidy
```
---
## 🚀 Uso

## 1) Um arquivo, sem webhook
```bash
go run . \
  -file /var/log/syslog \
  -pattern '(?i)error|critical' \
  -poll 300ms \
  -metrics-addr :9100
# /metrics disponível em http://localhost:9100/metrics
```
## 2) Múltiplos arquivos (glob), webhook Slack/Discord
```bash
go run . \
  -files "/var/log/nginx/*.log,/var/log/app/*.log" \
  -pattern '(?i)(error|critical|panic)' \
  -cooldown 30s -bundle-window 5s -bundle-max 20 \
  -webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
  -channel "alerts" \
  -title "[prod] nginx" \
  -metrics-addr :9100
```
## 3) Ler desde o início (como `tail -fn +1`)
```bash
go run . -file /var/log/syslog -from-start -pattern '(?i)error'

```
📋 Flags
| Flag             | Padrão         | Descrição                                                                                
| ---------------- | -------------- | ---------------------------------------------------------------------------------------- |
| `-file`          | `""`           | Caminho de um arquivo único.                                                             |
| `-files`         | `""`           | Lista de globs separados por vírgula (ex.: `"/var/log/nginx/*.log,/var/log/app/*.log"`). |
| `-pattern`       | `""`           | Regex para filtrar linhas (ex.: \`(?i)error critical\`). Se vazio, imprime todas as linhas.|
| `-from-start`    | `false`        | Lê desde o início do arquivo (senão segue do fim).                                       |
| `-poll`          | `300ms`        | Intervalo de polling quando não há novas linhas.                                         |
| `-cooldown`      | `30s`          | Janela para evitar spam de alertas por (arquivo+regex).                                  |
| `-bundle-window` | `5s`           | Janela de agregação antes de enviar o alerta.                                            |
| `-bundle-max`    | `20`           | Máximo de linhas por alerta.                                                             |
| `-metrics-addr`  | `""`           | Endereço para expor `/metrics` (ex.: `:9100`).                                           |
| `-webhook`       | `""`           | URL do Webhook (Slack/Discord). Vazio = sem envio.                                       |
| `-channel`       | `""`           | Nome do canal (opcional, útil no Slack).                                                 |
| `-title`         | `"Logwatcher"` | Título/Prefixo do alerta enviado.                                                        |

---

📈 Métricas Prometheus
- `logwatcher_lines_read_total{file}`
- `logwatcher_matches_total{file,pattern}`
- `logwatcher_alerts_sent_total{file,pattern}`
- `logwatcher_last_match_timestamp_seconds{file,pattern}`
- `logwatcher_active_targets`

Exemplo de `scrape_config`:
```yaml
scrape_configs:
  - job_name: 'logwatcher'
    static_configs:
      - targets: ['localhost:9100']
```
---
🧪 Testes manuais
Em um terminal:
```bash
go run . -file /var/log/syslog -pattern '(?i)error|critical' -metrics-addr :9100
```
Em outro:
```bash
echo "ERROR: database timeout" | sudo tee -a /var/log/syslog
```
Simule rotação:
```bash
sudo mv /var/log/syslog /var/log/syslog.1
sudo touch /var/log/syslog
sudo systemctl restart rsyslog || true
```
## 🛡️ Notas de segurança
- Para webhooks, mantenha a URL em segredos (env/CI) e evite expor em logs ou commits.
- Limite de bundle para evitar payloads muito grandes.
---
## 🗺️ Roadmap
- Suporte a templates de payload por provedor (Slack blocks, Discord embeds).
- Regras múltiplas (várias regex com ações distintas).
- Inputs de múltiplos diretórios recursivos.
- Persistência de offset (checkpoint) opcional.
