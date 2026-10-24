FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

FROM alpine:latest
WORKDIR /app
# Plantilla de variables de entorno (NO contiene secretos reales).
COPY --from=builder /app/.env.example ./.env.example
COPY --from=builder /app/server .
EXPOSE 3000
CMD ["./server"]
