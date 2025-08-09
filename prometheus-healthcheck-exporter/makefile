APP_NAME := healthcheck-exporter
IMAGE := $(APP_NAME):latest
PORT ?= 8080
URLS ?= https://example.com,https://httpbin.org/status/204,https://httpbin.org/status/500
INTERVAL ?= 30s
TIMEOUT ?= 3s
CONCURRENCY ?= 5

.PHONY: build run clean docker-build docker-run test

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(APP_NAME) ./main.go

run:
	./$(APP_NAME) -urls "$(URLS)" -interval $(INTERVAL) -timeout $(TIMEOUT) -concurrency $(CONCURRENCY) -port :$(PORT)

clean:
	rm -f $(APP_NAME)

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p $(PORT):8080 \
		$(IMAGE) \
		-urls "$(URLS)" -interval $(INTERVAL) -timeout $(TIMEOUT) -concurrency $(CONCURRENCY) -port :8080

test:
	go test ./...
