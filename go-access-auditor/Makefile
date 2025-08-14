APP=auditor

.PHONY: tidy build run docker up down logs test

tidy: ; go mod tidy

build:
	@mkdir -p bin
	go build -o bin/$(APP) ./cmd/server
	go build -o bin/agent ./cmd/agent

run:
	CONFIG_PATH=configs/config.yaml LOG_LEVEL=info ./bin/$(APP)

docker: ; docker build -t go-access-auditor:local .

up: ; docker compose up -d --build
down: ; docker compose down
logs: ; docker compose logs -f auditor

# Enviar evento de teste
send:
	echo 'rm -rf /' | ./bin/agent -api http://localhost:8080 -source bash -user $(USER)

PREFIX ?= /usr/local
BIN    ?= $(PREFIX)/bin

# Instala wrappers (requer mover binário original manualmente)
install-wrappers:
	install -m0755 scripts/kubectl-wrapper.sh $(BIN)/kubectl
	install -m0755 scripts/psql-wrapper.sh    $(BIN)/psql
	install -m0755 scripts/helm-wrapper.sh    $(BIN)/helm
	@echo "ATENÇÃO: renomeie os binários originais para *.real antes (kubectl.real, psql.real, helm.real)"

install-bash-hook:
	install -m0644 scripts/bash-history-hook.sh /etc/profile.d/auditor.sh
	@echo "Adicionado hook em /etc/profile.d/auditor.sh (reabra o shell)."

install-systemd-agent:
	install -m0755 bin/agent /usr/local/bin/auditor-agent
	install -d /etc/systemd/system
	install -m0644 packaging/systemd/auditor-agent.service /etc/systemd/system/auditor-agent.service
	systemctl daemon-reload
	systemctl enable --now auditor-agent.service

install-policies:
	install -d /etc/go-access-auditor
	install -m0644 policies/rules.extended.yaml /etc/go-access-auditor/rules.yaml
	@echo "Lembre-se de apontar configs/config.yaml para a política instalada."