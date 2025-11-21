# Etapa 1: build da aplicação em Go
FROM golang:1.22-alpine AS build

WORKDIR /app

# Copia módulos primeiro (cache de dependências)
COPY go.mod ./
RUN go mod download

# Copia o código-fonte
COPY . .

# Compila o binário
RUN go build -o healthchecker .

# Etapa 2: imagem final, pequena
FROM alpine:3.20

WORKDIR /app

# Certificados para HTTPS funcionar (https://...)
RUN apk add --no-cache ca-certificates

# Copia o binário já compilado
COPY --from=build /app/healthchecker /usr/local/bin/healthchecker

# Pasta padrão para o JSON
RUN mkdir -p /data

# ENTRYPOINT é o binário; flags vão no final do "docker run"
ENTRYPOINT ["healthchecker"]
