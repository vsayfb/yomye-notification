FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -o bin/notification-service ./cmd/cloudrun

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/bin/notification-service .

USER app

CMD ["./notification-service"]
