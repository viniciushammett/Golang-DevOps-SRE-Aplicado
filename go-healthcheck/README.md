<div align="center">
  <h1>🔎 HTTP Healthchecker (CLI)</h1>
  <p>Ferramenta de linha de comando em <b>Go</b> para checar a saúde de múltiplas URLs com concorrência, retries e saída JSON opcional</p>
  <img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey?style=flat-square" />
</div>

---

## 📖 Descrição

O **HTTP Healthchecker (CLI)** executa requisições HTTP/HTTPS para uma ou mais URLs, determina **UP/DOWN** (2xx/3xx = UP), mede **latência** (ms) e pode imprimir **JSON** por linha (NDJSON), ideal para **scripts**, **CI/CD** e **observabilidade leve**.

✨ Diferente de um exporter, **ele roda, imprime e finaliza** — perfeito para cronjobs e pipelines.

---

## ✨ Recursos

- ✅ Lista de URLs via `-urls` (separadas por vírgula)
- ⚡ Concorrência configurável (`-concurrency`)
- 🔁 Retries com backoff linear
- 🧾 Saída humana **ou** JSON por linha (`-json`)
- ⏱️ Timeout por request (fixo no código: 3s)
- 🧹 Exit code: `0` se todas UP, `1` se alguma falhar, `2` uso inválido

---

## 🛠 Instalação

```bash
git clone https://github.com/SEU_USUARIO/http-healthchecker.git
cd http-healthchecker
go build -o healthchecker .
```

Também pode rodar sem compilar:

```bash
go run . -urls https://example.com,https://httpbin.org/status/204
```
🚀 Uso
Básico 

```bash
./healthchecker -urls https://httpbin.org/status/204,https://httpbin.org/status/500
# [https://httpbin.org/status/204] UP 204 em 180ms
# [https://httpbin.org/status/500] DOWN 500 em 95ms
# echo $?  # <- 1 (teve falha)
```
JSON (NDJSON: 1 linha por URL)
```bash
./healthchecker -urls https://example.com,https://httpbin.org/status/500 -json
# {"url":"https://example.com","status":200,"ms":123,"up":true}
# {"url":"https://httpbin.org/status/500","status":500,"ms":87,"up":false}
```
Retries e concorrência
```bash
./healthchecker \
  -urls https://httpbin.org/status/503,https://httpbin.org/delay/1 \
  -retries 3 \
  -concurrency 5
```
📋 Flags
| Flag           | Tipo   | Padrão                | Descrição                                                              |
| -------------- | ------ | --------------------- | ---------------------------------------------------------------------- |
| `-urls`        | string | `https://example.com` | Lista de URLs separadas por vírgula.                                   |
| `-retries`     | int    | `0`                   | Tentativas extras em caso de erro/UP=false (backoff: 200ms, 400ms...). |
| `-json`        | bool   | `false`               | Imprime resultado em **JSON** (uma linha por URL).                     |
| `-concurrency` | int    | `5`                   | Máximo de checagens em paralelo.                                       |

```bash
Validação: URLs devem começar com http:// ou https://.
Timeout: 3s por request (ajustável no código, se quiser virar flag é simples).
```
🔚 Códigos de saída
- 0 – todas as URLs UP
- 1 – pelo menos uma URL DOWN ou erro de rede/timeout
- 2 – uso inválido (ex.: URLs vazias, esquema inválido)
- Esses códigos permitem gatear stages em CI/CD.

🧪 Exemplos para CI
Bash + jq (parse do JSON)
```bash
set -euo pipefail
./healthchecker -urls "https://example.com,https://httpbin.org/status/500" -json \
  | jq -r 'select(.up==false) | "\(.url) DOWN status=\(.status) ms=\(.ms)"' \
  && echo "Algumas URLs falharam" && exit 1 || true
```
GitHub Actions (job falha se alguma URL estiver DOWN)

```yaml
- name: Healthcheck
  run: |
    go build -o healthchecker .
    ./healthchecker -urls "https://example.com,https://httpbin.org/status/200" -retries 2
```
🧠 Como funciona (visão técnica)
- **Validação:** `net/url.ParseRequestURI` garante formato e esquema `http/https`.
- **Checagem:** `net/http.Client{Timeout: 3s}` + `GET`.
- **UP/DOWN:** `2xx` ou `3xx` ⇒ `UP=true`; demais ⇒ `UP=false`.
- **Latência:** `time.Since(start).Milliseconds()`.
- **Concorrência:** goroutines por URL + `sync.WaitGroup` + **semáforo** (`chan struct{}{}`, cap=concurrency).
- **Retries:** laço simples com backoff linear `200ms * tentativa`.

🧩 Saída JSON
```json
{
  "url": "https://example.com",
  "status": 200,
  "ms": 123,
  "up": true,
  "error": ""   // presente apenas quando falha por erro de rede/timeout
}

```
## 🐞 Troubleshooting

- **URL inválida:** `(use http(s)://)`  
  ➡ Adicione `http://` ou `https://` na frente da URL.

- **Timeouts frequentes:**  
  ➡ O serviço pode estar lento; aumente o `Timeout` no código ou reduza `-concurrency`.

- **Saída fora de ordem:**  
  ➡ Em modo concorrente, a ordem de impressão varia — normal.

## 🗺️ Roadmap

- `-timeout` como flag (em vez de valor fixo).
- **Headers múltiplos** (`-H "Name: Value"`) usando `http.NewRequest`.
- **Resumo final** (percentual `UP`, `p95` ms).
- Suporte a **input por arquivo** (`-file urls.txt`).
- Saída **JUnit/JSON** de suíte para **CI**.
