# 🔎 HTTP Healthchecker & Prometheus Exporter  
> Monitoramento simples, poderoso e educativo — escrito em Go.

<p align="center">
  <img src="https://img.shields.io/badge/Language-Go-blue?style=for-the-badge" />
  <img src="https://img.shields.io/badge/Observability-Prometheus%20%7C%20Grafana-orange?style=for-the-badge" />
  <img src="https://img.shields.io/badge/Runs-Docker%20%7C%20Baremetal-lightgrey?style=for-the-badge" />
  <img src="https://img.shields.io/badge/Mode-CLI%20%7C%20Exporter-green?style=for-the-badge" />
</p>

---

# 📚 Sumário
- [📖 Visão Geral](#-visão-geral)
- [🚀 Uso local (sem Docker)](#-uso-local-sem-docker)
  - [1️⃣ Clonar e Compilar](#1️⃣-clonar-e-compilar)
  - [2️⃣ Modo CLI Interativo](#2️⃣-modo-cli-interativo)
  - [3️⃣ Modo CLI Não Interativo](#3️⃣-modo-cli-não-interativo)
  - [4️⃣ Estrutura do JSON](#4️⃣-estrutura-do-json)
- [🔧 Flags Disponíveis](#-flags-disponíveis)
- [🧠 Arquitetura Interna](#-arquitetura-interna)
  - [health.go](#healthgo)
  - [metrics.go](#metricsgo)
- [🐳 Docker](#-docker)
- [📊 Stack Completa: Prometheus + Grafana](#-stack-completa-prometheus--grafana)
  - [Configuração do Grafana](#configuração-do-grafana)
  - [Painéis Prontos](#painéis-prontos)
- [✨ Roadmap Evolutivo](#-roadmap-evolutivo)
- [📄 Licença](#-licença)

---

# 📖 Visão Geral

Este projeto demonstra, de forma clara e prática, como criar uma cadeia completa de observabilidade:

> **Go → Healthcheck → JSON → Prometheus → Grafana → Docker**

Com ele você aprende:

- Como implementar healthchecks reais  
- Como expor métricas customizadas  
- Como montar uma stack completa de observabilidade  
- Como integrar tudo com Docker, Prometheus e Grafana  
- Como transformar um simples programa Go em um **exporter profissional**

---

# 🚀 Uso local (sem Docker)

## 1️⃣ Clonar e Compilar

```bash
git clone https://github.com/SEU_USUARIO/go-healthcheck.git
cd go-healthcheck

go mod tidy
go build -o healthchecker .
```

---

## 2️⃣ Modo CLI Interativo

```bash
./healthchecker
```

Exemplo:

```
URL para healthcheck [Endereço ou Site]: google.com
Timeout em segundos [3]: 5
Arquivo de saída JSON [health.json]: resultados.json

✅ https://google.com saudável! (status 200, 120ms)
📁 Resultado adicionado em resultados.json
```

---

## 3️⃣ Modo CLI Não Interativo

```bash
./healthchecker \
  -interactive=false \
  -url=https://www.google.com \
  -timeout=5 \
  -out=health.json
```

Saída:

```
✅ https://www.google.com saudável! (status 200, 95ms)
📁 Resultado adicionado em health.json
```

---

## 4️⃣ Estrutura do JSON

```json
[
  {
    "url": "https://www.google.com",
    "status": "UP",
    "code": 200,
    "elapsed_ms": 95,
    "checked_at": "2025-11-20T23:21:00-03:00"
  },
  {
    "url": "https://www.google.com",
    "status": "DOWN",
    "code": 500,
    "elapsed_ms": 80,
    "checked_at": "2025-11-20T23:22:10-03:00"
  }
]
```

---

# 🔧 Flags Disponíveis

| Flag | Tipo | Padrão | Descrição |
|------|------|---------|-----------|
| `-url` | string | Endereço/Site | URL alvo do healthcheck |
| `-timeout` | int | 3 | Timeout por request |
| `-out` | string | health.json | Arquivo JSON de saída |
| `-interactive` | bool | true | Perguntas interativas |
| `-metrics` | bool | false | Ativa modo Prometheus Exporter |
| `-interval` | int | 15 | Loop de intervalos no modo métricas |
| `-listen` | string | :8080 | Porta do endpoint `/metrics` |

---

# 🧠 Arquitetura Interna

## `health.go`

```go
type HealthResult struct {
    URL       string    `json:"url"`
    Status    string    `json:"status"`
    Code      int       `json:"code"`
    ElapsedMS int64     `json:"elapsed_ms"`
    CheckedAt time.Time `json:"checked_at"`
}
```

### 🔍 Regras Principais
- HTTP com `http.Client{Timeout: ...}`
- Cálculo de latência com `time.Since(start)`
- UP = status 200–399  
- DOWN = erro ou HTTP 400+  
- Sempre retorna JSON consistente

---

## `metrics.go`

### 📡 Métricas Expostas

```
healthchecker_up{url="..."} = 0 ou 1
healthchecker_latency_ms{url="..."} = milissegundos
```

### 🔄 Funcionamento Interno
- Loop interno lendo `checkHTTP()`
- Atualiza `GaugeVec`
- Exposição via: `/metrics`

---

# 🐳 Docker

## Build

```bash
docker build -t healthchecker:2.0 .
```

---

## Modo CLI Salvando JSON no Host

```bash
mkdir -p data

docker run --rm \
  -v $(pwd)/data:/data \
  healthchecker:2.0 \
  -interactive=false \
  -url=https://www.google.com \
  -timeout=5 \
  -out=/data/health.json
```

---

## Modo Exporter (Prometheus)

```bash
docker run --rm \
  -p 8080:8080 \
  healthchecker:2.0 \
  -interactive=false \
  -url=https://www.google.com \
  -timeout=3 \
  -metrics=true \
  -interval=15 \
  -listen=:8080
```

📡 Acesse:  
http://localhost:8080/metrics

---

# 📊 Stack Completa: Prometheus + Grafana

O repositório inclui:

- `prometheus.yml`  
- `docker-compose.yml`  

## Subir os serviços

```bash
docker compose up
```

### Endpoints

- **Healthchecker** → http://localhost:8080/metrics  
- **Prometheus** → http://localhost:9090  
- **Grafana** → http://localhost:3000  
  - login: `admin` / `admin`

---

## Configuração do Grafana

1. Acesse **http://localhost:3000**
2. Vá em: **Connections → Data Sources → Add data source**
3. Escolha **Prometheus**
4. Configure:

```
URL: http://prometheus:9090
```

5. Clique em **Save & Test**

---

## Painéis Prontos

### 🔹 Status UP/DOWN

```promql
healthchecker_up{url="https://www.google.com"}
```

### 🔹 Latência (ms)

```promql
healthchecker_latency_ms{url="https://www.google.com"}
```

---

# 📄 Licença

MIT
