FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/alert-router ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/alert-router /usr/local/bin/alert-router
COPY configs/ /app/configs/
VOLUME ["/app/data"]
ENV CONFIG_PATH=/app/configs/config.yaml
ENV HTTP_ADDR=:8080
ENV LOG_LEVEL=info
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/alert-router"]