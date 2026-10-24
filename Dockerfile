# Stage 1: Build frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY frontend/ .
ARG VITE_API_URL=https://brydyan.me
RUN VITE_API_URL=$VITE_API_URL pnpm build

# Stage 2: Build backend
FROM golang:1.26-alpine AS backend-build
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY database/migrations ./database/migrations
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 3: Runtime
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend-build /app/main .
COPY --from=backend-build /app/database/migrations ./database/migrations
COPY --from=frontend-build /app/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./main"]
