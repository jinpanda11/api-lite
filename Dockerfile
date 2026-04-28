# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --silent
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /src/backend/web ./web
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o new-api-lite .

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app/data
COPY --from=backend-builder /src/new-api-lite /app/new-api-lite

ENV CONFIG_PATH=config.yaml
EXPOSE 3000

CMD ["/app/new-api-lite"]
