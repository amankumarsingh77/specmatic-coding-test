FROM golang:1.23 AS builder
WORKDIR /app/go-service

COPY go-service/go.mod go-service/go.sum ./
RUN go mod download

COPY go-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o service

FROM scratch
WORKDIR /
COPY --from=builder /app/go-service/service /service
COPY --from=builder /app/go-service/config.yaml /config.yaml

EXPOSE 8090

ENTRYPOINT ["/service"]
