FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/server

# Кросс-компиляция под linux/arm64 без CGO
ENV CGO_ENABLED=0 GOOS=linux GOARCH=arm64

RUN go build -o server ./cmd/main.go

# Минимальный образ без зависимостей
FROM scratch

WORKDIR /app
COPY --from=builder /app/server/server .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/server/migrations ./migrations
COPY --from=builder /app/server/static ./static


CMD ["./server"]
