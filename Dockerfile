FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o orderflow ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/orderflow .
COPY --from=builder /app/web ./web

EXPOSE 8080

CMD ["./orderflow"]