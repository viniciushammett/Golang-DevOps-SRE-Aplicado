<div align="center">
  <h1>📊 Service Status Dashboard</h1>
  <p>Aplicação em Go que exibe o status de múltiplos serviços HTTP em tempo real, com visualização em dashboard web.</p>
</div>

---

## 📖 Descrição

O **Service Status Dashboard** é uma ferramenta escrita em **Go** para monitorar a disponibilidade de múltiplos serviços HTTP.  
Ele realiza verificações periódicas e exibe os resultados em um dashboard web simples, podendo carregar a lista de serviços a partir de **YAML**, **JSON** ou **inline** via linha de comando.

Ideal para **DevOps/SRE**, **lab environments** e testes rápidos de disponibilidade.

---

## ✨ Recursos

- 🔄 Checagem periódica de múltiplos serviços.
- 📂 Suporte a configuração via:
  - Arquivo **YAML**
  - Arquivo **JSON**
  - Lista inline (`-services "Name,URL;Name,URL"`)
- 🌐 Dashboard web em tempo real.
- 🎨 Tema claro/escuro com toggle.
- 🐳 Deploy Docker pronto.
- ☸️ Manifests Kubernetes com `/metrics` para Prometheus.

---

### 🚀 Uso

### 1️⃣ Executar localmente com configuração inline
```bash
go run . -services "Google,https://www.google.com;GitHub,https://github.com" -interval 30s -port :8080
```

### 2️⃣ Usando arquivo YAML
### services.yaml

```yaml
services:
  - name: Google
    url: https://www.google.com
  - name: GitHub
    url: https://github.com
```
```bash
go run . -config services.yaml -interval 30s -port :8080
```
### 3️⃣ Usando arquivo JSON
### services.json

```json
[
  { "name": "Google", "url": "https://www.google.com" },
  { "name": "GitHub", "url": "https://github.com" }
]
```
```bash
go run . -config services.json -interval 30s -port :8080
```
##

### 🐳 Docker
### Build
```bash
docker build -t service-status-dashboard .
```
### Run
```bash
docker run -p 8080:8080 \
  -v $(pwd)/services.yaml:/app/services.yaml \
  service-status-dashboard \
  -config /app/services.yaml
```
##

### ☸️ Kubernetes + Prometheus
### Deployment + Service
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service-status-dashboard
spec:
  replicas: 1
  selector:
    matchLabels:
      app: service-status-dashboard
  template:
    metadata:
      labels:
        app: service-status-dashboard
    spec:
      containers:
        - name: dashboard
          image: service-status-dashboard:latest
          args:
            - -config=/config/services.yaml
            - -interval=30s
            - -port=:8080
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: config
              mountPath: /config
      volumes:
        - name: config
          configMap:
            name: service-status-dashboard-config
---
apiVersion: v1
kind: Service
metadata:
  name: service-status-dashboard
spec:
  selector:
    app: service-status-dashboard
  ports:
    - port: 8080
      targetPort: 8080
      name: http
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: service-status-dashboard
spec:
  selector:
    matchLabels:
      app: service-status-dashboard
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```
##
### 🛠 Flags
| Flag        | Default | Descrição                                      |
| ----------- | ------- | ---------------------------------------------- |
| `-services` | `""`    | Lista inline de serviços (`Name,URL;Name,URL`) |
| `-config`   | `""`    | Caminho para arquivo YAML ou JSON              |
| `-interval` | `30s`   | Intervalo entre verificações                   |
| `-port`     | `:8080` | Porta do servidor web                          |

##
### 🐞 Troubleshooting
- ### "arquivo YAML sem serviços" 
   O arquivo não contém a chave services ou está vazio.

- ### Timeouts frequentes 
  Serviço pode estar lento; aumente o timeout no código ou reduza o -interval.

- ### Erro de parsing no config 
  Verifique a sintaxe YAML/JSON.
