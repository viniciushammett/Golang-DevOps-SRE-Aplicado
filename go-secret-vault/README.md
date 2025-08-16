# Go Secret Vault (GSV)

**GSV** é um cofre de segredos em **Go** para estudo: criptografia **AES‑256‑GCM**, **JWT**, **API REST + CLI**, **TTL**, **Audit Log** e **export para Kubernetes Secrets**.

> ⚠️ Motivo Educacional e Aprendizagem, não substitui um Vault Seguro para produção.

## ✨ Recursos
- 🔐 Criptografia AES‑256‑GCM com HKDF
- 🔑 JWT (HS256) + users (bcrypt) via `config.yaml`
- 🧰 API REST + CLI (`gsv`)
- ⏱️ TTL e coleta automática (reaper)
- 📜 Audit logging JSONL
- ☸️ Export de Secret (YAML) para Kubernetes
- 🚪 (Opcional) Stubs para Transit API

## 🚀 Começo rápido
```bash
# 1) Gerar segredos (exemplo)
export GSV_MASTER_KEY=$(openssl rand -base64 32)
export GSV_JWT_SECRET=$(openssl rand -base64 32)

# 2) Rodar API (local)
go run ./cmd/api

# 3) Login + operações (CLI)
go run ./cmd/cli login -u admin -p admin
go run ./cmd/cli put api.key -v 'super-secret' --ttl 2h
go run ./cmd/cli ls
go run ./cmd/cli get <id>
go run ./cmd/cli export-k8s <id> --namespace default --key API_KEY > secret.yaml
```
**Via Docker**
```bash
docker compose -f deployments/docker-compose.yaml up --build
```
**Via Kubernetes**
```bash
kubectl apply -f deployments/k8s/configmap.yaml
kubectl create secret generic gsv-secrets \
  --from-literal=GSV_MASTER_KEY="$GSV_MASTER_KEY" \
  --from-literal=GSV_JWT_SECRET="$GSV_JWT_SECRET" -n default --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
```
##
### 🔌 API
Autenticação: `Authorization: Bearer <token>` após `POST /login.`
```bash
POST /login
{ "username": "admin", "password": "admin" } → 200 { "token": "..." }

POST /secrets
{ "name": "db.password", "value": "S3cr3t!", "ttl": "1h", "meta": {"owner":"devops"} } → 200 { "id": "...", "name": "db.password", ... }

GET /secrets
→ 200 [ { "id": "...", "name": "db.password", ... } ]

GET /secrets/{id}
→ 200 { "id": "...", "name": "db.password", "value": "S3cr3t!" }

PUT /secrets/{id}
{ "value": "n3w", "ttl": "2h", "meta": {"env":"prod"} }
→ 200 { ... }

DELETE /secrets/{id}
→ 204

POST /secrets/{id}/export/k8s
{ "namespace": "default", "key": "PASSWORD" }
→ 200 (YAML do Secret)
```
**cURL rápido**
```bash
# Login
TOKEN=$(curl -s -XPOST http://localhost:8080/login -d '{"username":"admin","password":"admin"}' -H 'Content-Type: application/json' | jq -r .token)

# Criar
curl -s -XPOST http://localhost:8080/secrets/ -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"db.password","value":"S3cr3t!","ttl":"1h","meta":{"team":"sre"}}'

# Listar
curl -s http://localhost:8080/secrets/ -H "Authorization: Bearer $TOKEN" | jq

# Obter
curl -s http://localhost:8080/secrets/<id> -H "Authorization: Bearer $TOKEN" | jq

# Exportar K8s
curl -s -XPOST http://localhost:8080/secrets/<id>/export/k8s -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"namespace":"default","key":"PASSWORD"}'
```
##
### 🔐 Segurança e boas práticas
- **Mestre key e JWT secret** via **env/Secret** (não comitar valores).
- Rotacione chaves e audite acessos (arquivo `audit.log`).
- Restrinja rede/ingresso ao serviço.
- Considere Hardened base images / distroless.
##
### 🧩 Integração CI/CD (exemplo conceitual)
```yaml
# .github/workflows/pipeline.yaml (trecho fictício)
steps:
  - name: Get secret from GSV
    env:
      GSV_TOKEN: ${{ secrets.GSV_TOKEN }}
    run: |
      SECRET_ID="..."
      VAL=$(curl -s http://gsv.example/secrets/$SECRET_ID -H "Authorization: Bearer $GSV_TOKEN" | jq -r .value)
      echo "db_password=$VAL" >> $GITHUB_OUTPUT
```
##
### ☸️ K8s — Exportar Secret
```bash
go run ./cmd/cli export-k8s <id> --namespace default --key PASSWORD > secret.yaml
kubectl apply -f secret.yaml
```
##
### 🧪 Teste rápido
```bash
curl -s http://localhost:8080/healthz
```
##
### 🧱 Frontend

Diretório `frontend/` inclui um mini dashboard (login + criar/listar segredos). Para rodar:
```bash
cd frontend
npm i
npm run dev
```

##
### 📜 Licença

MIT
