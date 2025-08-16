APP=lad

.PHONY: tidy build run docker up down logs send

tidy: ; go mod tidy

build:
	@mkdir -p bin
	go build -o bin/$(APP) ./cmd/server

run:
	CONFIG_PATH=configs/config.yaml LOG_LEVEL=info ./bin/$(APP)

docker: ; docker build -t go-log-anomaly-detector:local .

up: ; docker compose up -d --build
down: ; docker compose down
logs: ; docker compose logs -f lad

send:
	bash examples/send_sample.sh

TOOLS_BIN ?= bin

# build do exportador
build-export:
	@mkdir -p $(TOOLS_BIN)
	go build -o $(TOOLS_BIN)/export-logs ./tools/export_logs_csv.go

# exporta logs do BoltDB atual para CSV
export-logs: build-export
	./$(TOOLS_BIN)/export-logs -db data/log-anomaly.db -out ml/logs.csv
	@echo "CSV pronto em ml/logs.csv"

# fluxo completo de treino (gera stats do modelo)
train-ml: export-logs
	python ml/trainer.py --input ml/logs.csv --output ml/model.stats.json
	@echo "Modelo atualizado em ml/model.stats.json"