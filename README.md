<div align="center">
  <h1>🚀 Golang para DevOps/SRE Aplicado </h1>
  <p>Repositório de estudos e projetos práticos em <strong>Golang</strong> voltados para <strong>DevOps</strong> e <strong>SRE</strong>, aplicando conceitos reais de monitoramento, observabilidade e automação — com o diferencial de <strong>criar ferramentas próprias</strong>, mesmo diante de diversas soluções prontas no mercado.</p>
  
  <img src="https://res.cloudinary.com/practicaldev/image/fetch/s--fu79u6To--/c_limit%2Cf_auto%2Cfl_progressive%2Cq_auto%2Cw_880/https://github.com/kodelint/blog-assets/raw/main/images/02-learn-go.png" width="700"/>
  
  <!-- Badges -->
  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.24+-blue.svg?style=for-the-badge&logo=go" alt="Go Version"></a>
    <a href="https://github.com/viniciushammett/Golang-DevOps-SRE-Aplicado/stargazers"><img src="https://img.shields.io/github/stars/viniciushammett/Golang-DevOps-SRE-Aplicado?style=for-the-badge" alt="GitHub Stars"></a>
    <a href="https://github.com/viniciushammett/Golang-DevOps-SRE-Aplicado/issues"><img src="https://img.shields.io/github/issues/viniciushammett/Golang-DevOps-SRE-Aplicado?style=for-the-badge" alt="GitHub Issues"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green.svg?style=for-the-badge" alt="MIT License"></a>
  </p>
</div>

---

## 📌 Sobre o Repositório

Este repositório é um **laboratório prático** de projetos em Go, criados para aplicar conceitos de **DevOps** e **Site Reliability Engineering** no dia a dia.

Diferente de apenas consumir ferramentas prontas, o objetivo aqui é **desenvolver soluções sob medida** para cenários críticos, garantindo:

- **Controle total** sobre código e funcionalidades.
- **Portabilidade** (um único binário, sem dependências externas).
- **Performance** e baixo consumo de recursos.
- **Segurança e auditabilidade** do código.

Cada projeto é **100% funcional**, com código aberto, documentação e exemplos de uso, podendo ser adaptado para ambientes reais.

> 🎯 Objetivo: unir estudo prático + criação de ferramentas úteis para operação, monitoramento e automação, mostrando capacidade de **engenharia de soluções** e não apenas de operação.

---

## 📂 Projetos Disponíveis

| Projeto | Descrição | Recursos Principais | Link |
|---------|-----------|--------------------|------|
| 🩺 **Healthchecker** | CLI para verificar múltiplas URLs com concorrência. | Status HTTP, tempo de resposta, saída JSON, retries. | [📄 Leia mais](./go-healthcheck/README.md) |
| 💽 **Disk Usage Monitor** | Mostra uso de disco de um diretório. | Total, usado, livre, erros tratados. | [📄 Leia mais](./go-diskmonitor/README.md) |
| 📊 **Prometheus Healthcheck Exporter** | Exporter que expõe métricas HTTP. | UP/DOWN, latência, status code, deploy em Kubernetes. | [📄 Leia mais](./prometheus-healthcheck-exporter/README.md) |
| 🔍 **Release Checker API** | API para buscar última release de um repositório. | JSON output, integração CI/CD. | [📄 Leia mais](./release-checker-api/README.md) |
| 🧾 **Logwatcher** | Tail de logs com regex, rotação, múltiplos arquivos, deduplicação/cooldown, métricas Prometheus e webhook. | Regex, fsnotify, glob múltiplo, Prometheus, webhook. | [📄 Leia mais](./logwatcher/README.md) |
| 📡 **SRE Monitor** | Monitor HTTP minimalista em Go com métricas Prometheus e healthcheck. | Configuração via YAML, logs estruturados, integração com Prometheus + Grafana (dashboard incluso). | [📄 Leia mais](./go-sre-monitor/README.md) |
| 🌀 **K8s Pod Restarter** | CLI, API e Scheduler para reinício seguro de pods no Kubernetes. | Configuração via YAML, métricas Prometheus, integração com Grafana e RBAC mínimo. | [📄 Leia mais](./k8s-pod-restarter/README.md) |
| 🧩 **Go Log Aggregator** | Agregador de logs com tail em tempo real e API de busca. | Fontes: arquivo/HTTP/stdin, ring buffer, filtros regex, métricas Prometheus. | [📄 Leia mais](./go-log-aggregator/README.md) |
| 🚨 **Go Alert Router & Notifier** | Serviço Golang para receber, deduplicar, agrupar e rotear alertas para múltiplos canais com métricas Prometheus. | CLI + API + retries com backoff, silences, rate limit, integração com Slack, Email, PagerDuty e dashboards Grafana. | [📄 Leia mais](./go-alert-router/README.md) |
| 🚀 **Go Deploy Orchestrator** | Orquestrador de deploys e rollback automático para Kubernetes com integração CI/CD. | API + CLI, canary e blue-green, thresholds Prometheus, histórico e aprovações manuais. | [📄 Leia mais](./go-deploy-orchestrator/README.md) |
| 🛡 **Go Access Auditor** | Auditoria centralizada de acessos e comandos em ambientes críticos (Linux, Kubernetes, DBs). | Agente + API + Dashboard, alertas para comandos sensíveis, métricas Prometheus e relatórios CSV/PDF. | [📄 Leia mais](./go-access-auditor/README.md) |
| 🤖 **Go Log Anomaly Detector** | Pipeline em Go com tracing **OpenTelemetry**, detecção de anomalias com **ML** e **frontend React** para visualização em tempo real. | CI/CD com workflows de **MLOps**, suporte a Prometheus/Grafana e deploy em Docker Compose. | [📄 Leia mais](./go-log-anomaly-detector/README.md) |

---

## 🛠 Como Iniciar um Novo Projeto Go

```bash
# Criar pasta do projeto
mkdir novo-projeto && cd novo-projeto

# Inicializar o módulo Go
go mod init github.com/seuusuario/Golang-DevOps-SRE-Aplicado/novo-projeto

# Adicionar dependências
go get <pacote>

# Rodar o projeto
go run .
```
##
### 🤝 Contribuições
Contribuições são bem-vindas!
Sinta-se à vontade para sugerir melhorias, novas funcionalidades ou enviar PRs.
##
### 📜 Licença
Este repositório é licenciado sob a MIT License.
Consulte o arquivo LICENSE para mais informações.
