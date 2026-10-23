# Multi-stage build para frontend
# Stage 1: Build
FROM node:20-alpine as builder

WORKDIR /app

# Copiar package files
COPY package.json pnpm-lock.yaml ./

# Instalar dependencias
RUN npm install -g pnpm && pnpm install --no-frozen-lockfile

# Copiar código fuente
COPY . .

# Build arg para la URL del backend (default: localhost)
ARG VITE_API_URL=http://localhost:3000
ENV VITE_API_URL=$VITE_API_URL

# Buildear
RUN pnpm run build

# Stage 2: Serve
FROM node:20-alpine

WORKDIR /app

# Instalar serve: servidor Node.js con SPA fallback nativo
RUN npm install -g serve

# Copiar dist del stage anterior
COPY --from=builder /app/dist ./dist

# Exponer puerto
EXPOSE 5173

# Servir con fallback automático a index.html
CMD ["serve", "-s", "dist", "-l", "5173"]
