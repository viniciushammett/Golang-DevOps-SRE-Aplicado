<div align="center">
  <h1>🚀 Golang DevOps/SRE Aplicado</h1>
  <p>Repositório de estudos e projetos práticos em <strong>Golang</strong> voltados para <strong>DevOps</strong> e <strong>SRE</strong>, aplicando conceitos reais de monitoramento, observabilidade e automação.</p>
  
  <img src="https://res.cloudinary.com/practicaldev/image/fetch/s--fu79u6To--/c_limit%2Cf_auto%2Cfl_progressive%2Cq_auto%2Cw_880/https://github.com/kodelint/blog-assets/raw/main/images/02-learn-go.png" width="700"/>
  
  <!-- Badges -->
  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-blue.svg?style=for-the-badge&logo=go" alt="Go Version"></a>
    <a href="https://github.com/viniciushammett/Golang-DevOps-SRE-Aplicado/stargazers"><img src="https://img.shields.io/github/stars/viniciushammett/Golang-DevOps-SRE-Aplicado?style=for-the-badge" alt="GitHub Stars"></a>
    <a href="https://github.com/viniciushammett/Golang-DevOps-SRE-Aplicado/issues"><img src="https://img.shields.io/github/issues/viniciushammett/Golang-DevOps-SRE-Aplicado?style=for-the-badge" alt="GitHub Issues"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green.svg?style=for-the-badge" alt="MIT License"></a>
  </p>
</div>

---

## 📌 Sobre o Repositório
Este repositório é um **laboratório prático** de projetos em Go, criados para aplicar conceitos de **DevOps** e **Site Reliability Engineering** no dia a dia.  
Cada projeto aqui é **100% funcional**, com código aberto, documentação e exemplos de uso, podendo ser adaptado para ambientes reais.

> 🛠 Objetivo: unir estudo prático + criação de ferramentas úteis para operação e monitoramento.

---

## 📂 Projetos Disponíveis

| Projeto | Descrição | Recursos Principais | Link |
|---------|-----------|--------------------|------|
| 🩺 **Healthchecker** | CLI para verificar múltiplas URLs com concorrência. | Status HTTP, tempo de resposta, saída JSON, retries. | [📄 Leia mais](./healthchecker/README.md) |
| 💽 **Disk Usage Monitor** | Mostra uso de disco de um diretório. | Total, usado, livre, erros tratados. | [📄 Leia mais](./disk-usage-monitor/README.md) |
| 📊 **Prometheus Healthcheck Exporter** | Exporter que expõe métricas HTTP. | UP/DOWN, latência, status code, deploy em Kubernetes. | [📄 Leia mais](./prometheus-healthcheck-exporter/README.md) |
| 🔍 **Release Checker API** | API para buscar última release de um repositório. | JSON output, integração CI/CD. | [📄 Leia mais](./release-checker-api/README.md) |

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
🤝 Contribuições
Sinta-se à vontade para sugerir melhorias ou enviar PRs.

📜 Licença
Este repositório é licenciado sob a MIT License.
Consulte o arquivo LICENSE para mais informações.
