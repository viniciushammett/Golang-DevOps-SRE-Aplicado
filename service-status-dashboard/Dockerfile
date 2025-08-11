# ---------- builder ----------
FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata git

# deps
COPY go.mod go.sum ./
RUN go mod download

# código
COPY . .

# build estático
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w -extldflags=-static" -o /out/ssd .

# ---------- runner (distroless, não-root) ----------
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
# binário
COPY --from=builder /out/ssd /app/ssd
# assets estáticos (dashboard)
COPY static/ /app/static/
# entrypoint
USER nonroot:nonroot
ENTRYPOINT ["/app/ssd"]