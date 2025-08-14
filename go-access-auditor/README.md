# 🛡 Go Access Auditor

Auditoria centralizada de **acessos e comandos** (Linux/Kubernetes/DBs). Inclui **Agente**, **API/Coletor**, **Dashboard HTMX**, **Métricas Prometheus**, **Export CSV** e **alertas Slack** para comandos sensíveis.

[![Full CI](https://github.com/viniciushammett/go-access-auditor/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
[![Policy QA](https://github.com/viniciushammett/go-access-auditor/actions/workflows/policy-qa.yml/badge.svg)](.github/workflows/policy-qa.yml)

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
**Início rápido**
```bash
go mod tidy
make build
CONFIG_PATH=configs/config.yaml ./bin/auditor
# em outro terminal:
echo 'kubectl delete pods --all -n prod' | ./bin/agent -api http://localhost:8080 -source kubectl -user $USER
```
**Configuração** (`configs/config.yaml`)
```yaml
server: { addr: ":8080" }
authToken: ""              # opcional p/ exigir Bearer no POST /v1/events
storage: { path: "data/auditor.db" }
rules:
  - { name: "danger-rm-root", regex: "(?i)\\brm\\s+-rf\\s+/(\\s|$)" }
  - { name: "drop-database",  regex: "(?i)\\bdrop\\s+database\\b" }
slack: { enabled: false, webhook: "https://hooks.slack.com/services/..." }
```
**API/Consultas**
- `POST /v1/events` (JSON): `{ when?, user, host?, source, command, meta? }`
- `GET /v1/events?q=drop&user=alice&source=psql&limit=200&sensitive=true`
- `GET /v1/export.csv` (mesmos parâmetros)
- `GET /metrics, GET /healthz`
##
**Docker Compose**
```bash
docker compose up --build
# API: http://localhost:8080  Grafana: http://localhost:3000  Prometheus: http://localhost:9090
```
##
**Dashboard**
Importe `dashboards/grafana-access-auditor.json` no Grafana (métricas de ingestão e matches sensíveis).
##
**Integração real (idéias)**
- **bash:** `PROMPT_COMMAND='history -a; history 1 | cut -c 8- | auditor-agent -api http://auditor:8080 -source bash -user $USER -cmd "$(history 1 | cut -c 8-)"'`
- **kubectl** wrapper: renomeie `kubectl` real e crie script que registra e delega.
- **auditd/snoopy:** encaminhar para stdin do agente.
##
### 🎛 Perfis de políticas

Você pode escolher entre três perfis prontos:

- `policies/rules.prudent.yaml` — foco em alto risco e baixo ruído (recomendado em produção).
- `policies/rules.extended.yaml` — conjunto abrangente “default”.
- `policies/rules.aggressive.yaml` — cobertura máxima, pode gerar mais alertas (sandbox/forense).

Para validar as políticas com exemplos:
```bash
make rules-validate-prudent
make rules-validate-aggressive
# opcional
make rules-validate-extended
```
Para testes unitários do motor de regras:
```bash
make rules-unit-test

Dica: mantenha seus exemplos reais (anonimizados) em policies/examples/*.jsonl para evitar regressões ao atualizar regex.

```
Config rápida para alternar o perfil

No `configs/config.yaml`, você pode “incluir” o conteúdo de um perfil usando `yq` no pipeline de build/deploy, ou simplesmente trocar manualmente. Ex.:

```bash
# usar perfil prudente
cp policies/rules.prudent.yaml configs/rules.yaml
# e no configs/config.yaml, deixe:
# rules: (conteúdo do rules.yaml) – ou importe via pipeline
```
##
### **Segurança**
- Habilite `authToken` para POSTs.
- Restrinja IPs ou use Ingress com Autenticação.
- Evite enviar dados sensíveis em claro.
##
### **Scripts de coleta (wrappers + hook Bash)**
Requisitos dos wrappers: `jq` e `curl`. Você pode embutir JSON sem jq, mas fica mais verboso.
## 
### 🔌 Wrappers & Hook Bash

- **Hook Bash**: adicione ao `~/.bashrc` (ou `/etc/profile.d/`):
```bash
source ./scripts/bash-history-hook.sh
export AUDITOR_API="http://auditor:8080"
export AUDITOR_TOKEN=""  # se usar auth
```
- **kubectl wrapper:**
```bash
sudo mv /usr/local/bin/kubectl /usr/local/bin/kubectl.real
sudo install -m0755 ./scripts/kubectl-wrapper.sh /usr/local/bin/kubectl
```
- **psql & helm: repita o procedimento trocando os nomes.**
- Dica: use `make install-wrappers` para instalar todos (renomeie os binários originais para *.real antes).
##
### 📋 Políticas de Regras (regex)
Use nossa política ampliada:
```bash
cp policies/rules.extended.yaml configs/
# e aponte em configs/config.yaml (ou mescle)
```
##
### 🧷 Agent como Daemon
- Kubernetes: `kubectl apply -f deploy/agent-daemonset.yaml`
- Linux: `make install-systemd-agent` (usa `packaging/systemd/auditor-agent.service`)
##
### 🔄 CI/CD

Este repositório vem com uma esteira **Full CI** no GitHub Actions cobrindo **lint**, **build**, **testes**, e **QA de políticas (regex)**.

[![Full CI](https://github.com/viniciushammett/go-access-auditor//actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
[![Policy QA](https://github.com/viniciushammett/go-access-auditor/actions/workflows/policy-qa.yml/badge.svg)](.github/workflows/policy-qa.yml)

> Ajuste `your-org/your-repo` para o caminho do seu repositório.

### 🧭 Resumo do pipeline

**Workflow:** `.github/workflows/ci.yml`  
**Gatilhos:** `push` e `pull_request` para `main`/`master`.

**Ordem dos jobs:**

1. **Lint** — `golangci-lint` em `./...`  
2. **Build** — compila **server** e **agent**  
3. **Tests** — executa `go test ./... -cover`  
4. **Policy QA** — valida regras (prudente/agressiva/extended) com exemplos + roda `go test` do pacote `internal/rules`

Se preferir granular, há também o **workflow dedicado** de políticas: `.github/workflows/policy-qa.yml` (roda em mudanças sob `policies/**`, `internal/rules/**`, etc.)

---

### 📦 O que cada job faz

- **Lint (golangci-lint)**  
  Padrão de qualidade e estilo. Atualize a versão do linter no workflow quando necessário.

- **Build**  
  Gera binários:
  - `bin/go-access-auditor` (server)
  - `bin/go-access-agent` (agent)

- **Tests**  
  Executa testes de todas as pastas (`./...`) com **coverage**.

- **Policy QA**  
  - Compila o utilitário `rules-tester` (`tools/rules_tester.go`).
  - Valida:
    - `policies/rules.prudent.yaml` com `policies/examples/prudent.jsonl`
    - `policies/rules.aggressive.yaml` com `policies/examples/aggressive.jsonl`
    - `policies/rules.extended.yaml` com os exemplos do agressivo (como sanity check)
  - Executa `go test` no pacote `internal/rules`.

> Dica: mantenha **exemplos reais (anonimizados)** em `policies/examples/*.jsonl` para evitar regressões nas regex.

---

### 🧪 Rodando localmente (antes do PR)

```bash
# 1) Lint (instale o linter se ainda não tiver)
golangci-lint run ./...

# 2) Build rápido
go build -o bin/auditor ./cmd/server
go build -o bin/agent  ./cmd/agent

# 3) Testes
go test ./... -v -cover

# 4) Validação de políticas
make rules-validate-prudent
make rules-validate-aggressive
# opcional
make rules-validate-extended
```
S
Se não tiver golangci-lint, instale:
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.59.1
```
