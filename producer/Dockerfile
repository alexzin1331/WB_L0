FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/producer

ENV CGO_ENABLED=0 GOOS=linux GOARCH=arm64

RUN go build -o producer ./cmd/main.go

FROM scratch

WORKDIR /app
COPY --from=builder /app/producer/producer .

CMD ["./producer"]
