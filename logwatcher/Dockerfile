# ---------- builder ----------
FROM golang:1.22-alpine AS builder
WORKDIR /src

# deps básicas
RUN apk add --no-cache ca-certificates tzdata git

# copie go.mod/go.sum primeiro para cache
COPY go.mod go.sum ./
RUN go mod download

# agora o código
COPY . .

# build estático, sem CGO
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w -extldflags=-static" -o /out/logwatcher .

# ---------- runner (distroless, não-root) ----------
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/logwatcher /app/logwatcher
USER nonroot:nonroot
ENTRYPOINT ["/app/logwatcher"]