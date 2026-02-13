# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/summithub ./cmd/api

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/summithub ./summithub

EXPOSE 8080

ENTRYPOINT ["/app/summithub"]
