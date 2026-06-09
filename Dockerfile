FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
LABEL org.opencontainers.image.source="https://github.com/huuthangle/ws-relay-go"
LABEL org.opencontainers.image.description="Lightweight Go websocket relay server"
EXPOSE 5001
CMD ["./server"]
