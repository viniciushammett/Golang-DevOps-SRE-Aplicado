# 🧩 **go-diskmonitor — Disk Usage Monitor em Go**

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![OS](https://img.shields.io/badge/OS-Linux%20%7C%20Windows-blue?logo=linux&logoColor=white)](#)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?logo=prometheus&logoColor=white)](https://prometheus.io)
[![Grafana](https://img.shields.io/badge/Grafana-Dashboard-F46800?logo=grafana&logoColor=white)](https://grafana.com)
[![Status](https://img.shields.io/badge/Status-Stable-brightgreen)](#)
[![License](https://img.shields.io/badge/License-MIT-yellow)](LICENSE)

**Monitor de uso de disco cross-platform com análise inteligente, exportação de métricas e integração real com Prometheus + Grafana.**

</div>

---

# 📚 **Sumário**
- [📘 Sobre o Projeto](#-sobre-o-projeto)
- [🛠️ Funcionalidades](#️-funcionalidades)
- [📁 Estrutura do Projeto](#-estrutura-do-projeto)
- [🏗️ Como Rodar](#️-como-rodar)
- [📊 Exportação para Prometheus](#-exportação-para-prometheus)
- [📈 Dashboard no Grafana](#-dashboard-no-grafana)
- [📄 Licença](#-licença)

---

# 📘 **Sobre o Projeto**

O **go-diskmonitor** é um monitor profissional de disco escrito em Go, compatível com **Linux** e **Windows**, capaz de:

- Ler informações reais do filesystem
- Identificar diretórios críticos que ocupam mais espaço
- Sugerir limpeza e executar limpeza automática
- Exportar métricas diretas para **Prometheus**
- Integrar com dashboards avançados no **Grafana**
- Simular “disco cheio” em ambiente controlado para demonstração

Este projeto compõe o **Projeto 2** do curso **DevOps/SRE em Go**.

---

# 🛠️ **Funcionalidades**

✔ Compatível com Linux e Windows  
✔ Monitora qualquer caminho (`/`, `/var`, `C:\`)  
✔ Formatação humana (MB, GB, TB)  
✔ Detecção de hotspots automáticos  
✔ Threshold configurável (default: 80%)  
✔ Limpeza segura de diretórios temporários  
✔ Exporta métricas para Prometheus (textfile collector)  
✔ Integração total com node_exporter  
✔ Dashboard Grafana pronto  
✔ Scripts para gerar e limpar dados fake  

---

# 📁 **Estrutura do Projeto**

```bash
go-diskmonitor/
│
├── main.go                 # Core do projeto
├── disk_unix.go            # Funções específicas Linux
├── disk_windows.go         # Funções específicas Windows
│
├── generate_demo_data.sh   # Script para simulação (encher disco)
├── cleanup_demo_data.sh    # Script para limpeza da simulação
│
└── README.md               # Este documento
```
---
# 🏗️ **Como Rodar**

🔹 Build Linux
```bash
go build -o go-diskmonitor .
```
🔹 Build Windows
```bash
GOOS=windows GOARCH=amd64 go build -o go-diskmonitor.exe .
```
🔹 Execução básica
```bash
./go-diskmonitor -path / -threshold 80
```
🔹 Exportando para Prometheus
```bash
./go-diskmonitor \
  -path / \
  -threshold 80 \
  -prom-file /var/lib/node_exporter/diskmonitor.prom
```
# 📊 Exportação para Prometheus

O arquivo .prom é gerado assim:
```bash
disk_usage_percent{mount="/"} 72.34
disk_free_user_bytes{mount="/"} 124812374
```
🔹 Habilitar o textfile collector
```bash
./node_exporter \
  --collector.textfile.directory=/var/lib/node_exporter
```
Acesse:
```bash
http://SEU-IP:9100/metrics
```
E pesquise:
```bash
disk_usage_percent

disk_free_user_bytes
```
# 📈 Dashboard no Grafana
1️⃣ Adicionar Prometheus
```bash
Configuration → Data Sources → Add Prometheus

URL: http://localhost:9090
```
2️⃣ Queries principais

📌 Percentual usado:
```bash
disk_usage_percent{mount="/"}
```

📌 Espaço livre:
```bash
disk_free_user_bytes{mount="/"}
```
3️⃣ Painéis sugeridos
```bash
Gauge → uso de disco

Graph → histórico

Stat → bytes livres

Alert → threshold > 80%
```
---
# 📄 Licença

Distribuído sob a licença MIT.
