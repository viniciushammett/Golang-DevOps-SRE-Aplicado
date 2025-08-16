FROM golang:1.22 as build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /gsv ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /
COPY --from=build /gsv /gsv
COPY configs/config.yaml /configs/config.yaml
ENV GSV_CONFIG=/configs/config.yaml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/gsv"]