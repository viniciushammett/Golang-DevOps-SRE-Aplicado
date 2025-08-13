FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.version=${VERSION:-0.1.0} -X main.commit=${COMMIT:-dev} -X main.date=$(date -u +%Y-%m-%d)" \
    -o /out/pod-restarter ./cmd/restarter

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/pod-restarter /usr/local/bin/pod-restarter
COPY configs/ /app/configs/
ENV LOG_LEVEL=info
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/pod-restarter"]