FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/deploy-orchestrator ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/deploy-orchestrator /usr/local/bin/deploy-orchestrator
COPY configs/ /app/configs/
ENV CONFIG_PATH=/app/configs/config.yaml
ENV HTTP_ADDR=:8080
ENV LOG_LEVEL=info
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/deploy-orchestrator"]