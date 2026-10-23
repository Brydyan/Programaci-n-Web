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
FROM nginx:alpine

WORKDIR /app

# Copiar configuración de Nginx para SPA
RUN echo 'server { \
  listen 5173; \
  root /app/dist; \
  index index.html; \
  location / { \
    try_files $uri $uri/ /index.html; \
  } \
}' > /etc/nginx/conf.d/default.conf

# Copiar dist del stage anterior
COPY --from=builder /app/dist ./dist

# Exponer puerto
EXPOSE 5173

# Iniciar Nginx
CMD ["nginx", "-g", "daemon off;"]
