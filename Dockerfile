FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o bin/notification-service ./cmd/notification

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/bin/notification-service .

CMD ["./notification-service"]