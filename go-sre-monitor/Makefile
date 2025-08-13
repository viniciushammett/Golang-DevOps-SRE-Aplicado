APP=go-sre-monitor
PKG=github.com/viniciushammett/$(APP)

.PHONY: build run test tidy lint docker

build:
	@echo ">> build"
	@go build -o bin/$(APP) ./cmd/monitor

run:
	CONFIG_PATH=configs/services.yaml HTTP_ADDR=:8080 LOG_LEVEL=info PROBE_INTERVAL=15s go run ./cmd/monitor

test:
	go test ./...

tidy:
	go mod tidy

docker:
	docker build -t $(APP):local .

compose-up:
	docker compose up -d

compose-down:
	docker compose down
