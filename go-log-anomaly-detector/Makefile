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