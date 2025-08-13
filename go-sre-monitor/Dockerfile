FROM golang:1.22 as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/go-sre-monitor ./cmd/monitor

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/go-sre-monitor /app/go-sre-monitor
COPY configs/ /app/configs/
ENV CONFIG_PATH=/app/configs/services.yaml
ENV HTTP_ADDR=:8080
ENV LOG_LEVEL=info
ENV PROBE_INTERVAL=15s
EXPOSE 8080
ENTRYPOINT ["/app/go-sre-monitor"]