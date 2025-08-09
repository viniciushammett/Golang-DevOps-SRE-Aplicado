<div align="center">
  <h1>📊 Monitor Disk Usage</h1>
  <p>Ferramenta de linha de comando em <b>Go</b> para exibir estatísticas detalhadas de uso de disco em sistemas Unix-like</p>
  <img src="https://img.shields.io/badge/platform-linux%20%7C%20macos-lightgrey?style=flat-square" />
</div>

---

## 📖 Descrição

O **Monitor Disk Usage** é uma **CLI (Command Line Interface)** desenvolvida em Go que coleta e exibe estatísticas do uso de disco para um diretório ou arquivo específico.  
As informações incluem:

- **Espaço total** do filesystem.
- **Espaço usado** (considerando blocos reservados).
- **Espaço livre para o usuário** (desconsiderando blocos reservados ao root).
- **Percentual de uso**.

⚙️ Internamente, a ferramenta utiliza `syscall.Statfs` para obter métricas diretamente do kernel, garantindo precisão e baixo overhead.

---

## ✨ Recursos

- Aceita **diretórios** ou **arquivos** como caminho de análise.
- Argumentos via **flag** (`-path`) ou **posicional**.
- Saída **legível** (formato IEC — GiB, MiB, etc.).
- Opção para exibir **bytes crus** (`-human=false`).
- Tratamento de erros com mensagens claras e códigos de saída adequados.
- Compatível com **Linux** e **macOS** (via build tags).

---

## 🛠 Instalação

```bash
# Clonar repositório
git clone https://github.com/SEU_USUARIO/monitor-disk-usage.git
cd monitor-disk-usage

# Compilar binário
go build -o disk-usage
