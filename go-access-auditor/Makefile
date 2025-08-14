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