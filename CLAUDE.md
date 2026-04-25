# new-api-lite

## Build
- `make all` — builds frontend (to backend/web/) + backend binary
- `make frontend` — builds frontend only
- `make build` — builds backend binary only (requires frontend build in backend/web/)

## Dev
- `make dev` — starts backend in dev mode (port 3000)
- `cd frontend && npm run dev` — frontend dev server (port 5173, proxies /api → :3000)

## Deploy (VPS)
- `sudo ./deploy.sh` — pull latest, build, restart service
- Service: `new-api-lite.service` (systemd template, adjust paths as needed)

## Config
- `backend/config.yaml.example` — template (committed)
- `backend/config.yaml` — runtime config (gitignored, copy from example)
- Edit JWT secret, SMTP, admin credentials before production use

## Architecture
- Go/Gin backend (SQLite via GORM, JWT auth)
- React/TypeScript frontend (Semi UI, zustand, React Router)
- Frontend static files embedded in Go binary via `//go:embed`
- Single binary deployment: `backend/new-api-lite` serves both API and frontend
