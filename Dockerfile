# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/server ./cmd/server

FROM alpine:3.20
WORKDIR /app

COPY --from=builder /bin/server /usr/local/bin/server
COPY .env.example /app/.env.example

EXPOSE 8080

CMD ["server"]
