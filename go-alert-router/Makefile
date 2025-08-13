APP=alert-router

.PHONY: tidy build run docker up down logs fmt test

tidy:
	@go mod tidy

build:
	@mkdir -p bin
	@go build -o bin/$(APP) ./cmd/server

run:
	CONFIG_PATH=configs/config.yaml HTTP_ADDR=:8080 LOG_LEVEL=info ./bin/$(APP)

docker:
	docker build -t go-alert-router:local .

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

fmt:
	go fmt ./...

test:
	go test ./...