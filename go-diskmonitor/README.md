<div align="center">
  <h1>📊 Monitor Disk Usage</h1>
  <p>Ferramenta de linha de comando em Go para exibir estatísticas de uso de disco de um diretório</p>
</div>

---

## 📖 Descrição
O **Monitor Disk Usage** é uma ferramenta de linha de comando que exibe estatísticas de uso de disco para um diretório especificado, incluindo:

- Espaço **total**
- Espaço **usado**
- Espaço **livre**

Inclui tratamento de erros para caminhos inválidos.

---

## ✨ Recursos
- Recebe o caminho do diretório como argumento na linha de comando.
- Saída no formato legível por humanos.
- Tratamento de erros para diretórios inexistentes ou sem permissão.

---

## 🛠 Uso
```bash
go run disk_usage.go /caminho/do/diretorio

Directory: /home/user
Total: 500 GB
Used: 312 GB
Free: 188 GB
```
