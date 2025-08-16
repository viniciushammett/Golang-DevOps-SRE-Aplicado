# Go Secret Vault (GSV)

**GSV** é um cofre de segredos em **Go** para estudo: criptografia **AES‑256‑GCM**, **JWT**, **API REST + CLI**, **TTL**, **Audit Log** e **export para Kubernetes Secrets**.

> ⚠️ Educacional; não substitui o Vault em produção.

## ✨ Recursos
- 🔐 Criptografia AES‑256‑GCM com HKDF
- 🔑 JWT (HS256) + users (bcrypt) via `config.yaml`
- 🧰 API REST + CLI (`gsv`)
- ⏱️ TTL e coleta automática (reaper)
- 📜 Audit logging JSONL
- ☸️ Export de Secret (YAML) para Kubernetes
- 🚪 (Opcional) Stubs para Transit API

## 🚀 Comece rápido
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