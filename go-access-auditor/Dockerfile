FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auditor ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agent ./cmd/agent

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/auditor /usr/local/bin/auditor
COPY --from=build /out/agent /usr/local/bin/auditor-agent
COPY configs/ /app/configs/
ENV CONFIG_PATH=/app/configs/config.yaml
ENV LOG_LEVEL=info
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/auditor"]