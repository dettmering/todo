FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .
RUN go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/todo ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 app \
    && mkdir -p /data && chown app:app /data

ENV DB_PATH=/data/todo.db \
    LISTEN_ADDR=:8080 \
    TZ=Europe/Berlin

USER app
WORKDIR /app
COPY --from=builder /out/todo /app/todo
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]
