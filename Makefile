.PHONY: dev build frontend clean tidy

# Start backend in dev mode
dev:
	cd backend && go run .

# Build backend binary
build:
	cd backend && go build -o new-api-lite .

# Build frontend and put output in backend/web/
frontend:
	cd frontend && npm install && npm run build

# Build everything
all: frontend build

# Tidy Go modules
tidy:
	cd backend && go mod tidy

# Remove build artifacts
clean:
	rm -f backend/new-api-lite
	rm -rf backend/web

# Run with race detector (dev)
race:
	cd backend && go run -race .
