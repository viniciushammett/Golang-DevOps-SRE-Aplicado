<div align="center">
  <h1>🚀 Golang DevOps/SRE Aplicado</h1>
  <p>Práticas e estudos aplicados de <b>DevOps</b> e <b>Site Reliability Engineering</b> utilizando a linguagem <b>Go (Golang)</b>.</p>
  <img src="https://res.cloudinary.com/practicaldev/image/fetch/s--fu79u6To--/c_limit%2Cf_auto%2Cfl_progressive%2Cq_auto%2Cw_880/https://github.com/kodelint/blog-assets/raw/main/images/02-learn-go.png" width="600"/>
</div>

---

## 📖 Sobre o Repositório
Este repositório reúne uma coleção de projetos **práticos** em Golang, criados para aplicar conceitos reais de DevOps e SRE no dia a dia.

A ideia é evoluir continuamente, adicionando novos módulos, explorando desde ferramentas simples de linha de comando até serviços complexos integrados com observabilidade, automação e cloud.

---

## 📂 Projetos Atuais

### 1️⃣ go-healthcheck
🔍 **Descrição:** Ferramenta de linha de comando para verificar a saúde (UP/DOWN) de múltiplas URLs, com suporte a concorrência, retries e saída JSON.  
📌 **Principais recursos:**
- Checagem concorrente de múltiplas URLs
- Saída legível ou em JSON
- Retentativas automáticas com backoff
- Validação de formato de URL (http/https)

📜 [Documentação detalhada](./go-healthcheck/README.md)

---

### 2️⃣ prometheus-healthcheck-exporter
📊 **Descrição:** Mini-exporter Prometheus escrito em Go para monitorar múltiplas URLs periodicamente, expondo métricas como UP/DOWN, latência e status HTTP.  
📌 **Principais recursos:**
- Métricas no formato Prometheus (`/metrics`)
- Configuração via flags
- Compatível com Kubernetes (Deployment + ServiceMonitor)
- Dockerfile seguro (distroless, non-root)

📜 [Documentação detalhada](./prometheus-healthcheck-exporter/README.md)

---

### 3️⃣ go-diskmonitor
💽 **Descrição:** Ferramenta CLI para exibir estatísticas de uso de disco de um diretório, incluindo espaço total, usado e livre, com tratamento de erros.  
📌 **Principais recursos:**
- Recebe caminho do diretório como argumento
- Saída human-readable
- Erro para caminhos inexistentes ou sem permissão

📜 [Documentação detalhada](./go-diskmonitor/README.md)

---

## 🛠️ Tecnologias Utilizadas
- **Golang** (concorrência com goroutines, canais e WaitGroups)
- **Prometheus Client Go**
- **Docker** (imagens distroless e non-root)
- **Kubernetes Manifests** (Deployment, Service, ServiceMonitor)
- **Markdown** para documentação

---

## 🌱 Roadmap Geral
- [ ] Adicionar testes unitários para todos os projetos
- [ ] Adicionar GitHub Actions para CI/CD
- [ ] Criar pacotes reutilizáveis internos (`/pkg`)
- [ ] Adicionar suporte a configuração via arquivo YAML/ENV
- [ ] Expandir para healthchecks TCP e ICMP
- [ ] Criar dashboards no Grafana para os exporters

---

## 🤝 Contribuindo
Contribuições são bem-vindas!  
Siga o fluxo padrão:
1. Fork do repositório
2. Criar branch (`feature/nome-funcionalidade`)
3. Commitar mudanças
4. Abrir um Pull Request

---

## 📜 Licença
Este projeto é licenciado sob a [MIT License](LICENSE).

---

<div align="center">
  <sub>Desenvolvido com 💙 para estudo e aplicação prática em DevOps e SRE usando Golang.</sub>
</div>
