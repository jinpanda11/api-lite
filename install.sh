#!/bin/bash
set -e

INSTALL_DIR="/opt/api-lite"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()   { echo -e "${RED}[ERROR]${NC} $1"; }

echo "=============================================="
echo "  new-api-lite Docker 一键安装"
echo "=============================================="
echo ""

if [ "$EUID" -ne 0 ]; then
  err "请以 root 运行: sudo bash install.sh"
  exit 1
fi

# ─── 1. Install Docker ───────────────────────────────────
if ! command -v docker &>/dev/null; then
  info "安装 Docker..."
  curl -fsSL https://get.docker.com | bash
  info "Docker 已安装"
else
  info "Docker 已装: $(docker --version)"
fi

if ! docker compose version &>/dev/null 2>&1; then
  info "安装 docker-compose-plugin..."
  apt-get update -qq && apt-get install -y -qq docker-compose-plugin
fi

# ─── 2. Stop old bare-metal service if exists ────────────
if systemctl is-active --quiet api-lite 2>/dev/null; then
  warn "检测到旧的 bare-metal 服务运行中，正在停止..."
  systemctl stop api-lite
  systemctl disable api-lite
  rm -f /etc/systemd/system/api-lite.service
  systemctl daemon-reload
fi

# ─── 3. Create directory structure ───────────────────────
mkdir -p "$INSTALL_DIR"

# ─── 4. Write docker-compose.yml ─────────────────────────
cat > "$INSTALL_DIR/docker-compose.yml" << 'DOCKEREOF'
services:
  api-lite:
    image: ghcr.io/jinpanda11/api-lite:latest
    restart: unless-stopped
    ports:
      - "3000:3000"
    volumes:
      - ./backend:/app/data
    environment:
      - GIN_MODE=release
DOCKEREOF

# ─── 5. Create default config if missing ─────────────────
if [ ! -f "$INSTALL_DIR/backend/config.yaml" ]; then
  mkdir -p "$INSTALL_DIR/backend"
  cat > "$INSTALL_DIR/backend/config.yaml" << 'CFGEOF'
server:
  port: 3000
  debug: false

database:
  driver: sqlite
  dsn: "new-api-lite.db"

jwt:
  secret: "change-me-to-a-random-string"
  expire_hours: 168

smtp:
  host: ""
  port: 465
  username: ""
  password: ""
  from: ""
  ssl: true

admin:
  username: "jinpanda"
  email: "admin@jinpanda.com"
  password: "s1059416282"
CFGEOF
  info "已创建 config.yaml，请修改 JWT secret 等参数: vi $INSTALL_DIR/backend/config.yaml"
else
  info "config.yaml 已存在，跳过"
fi

# ─── 6. Migrate old DB if exists ─────────────────────────
# Old bare-metal DB might be at /opt/api-lite/backend/new-api-lite.db
# Docker mounts ./backend to /app/data, so DB is already in the right place

# ─── 7. Pull and start ───────────────────────────────────
info "拉取 Docker 镜像..."
cd "$INSTALL_DIR"
docker compose pull

info "启动服务..."
docker compose up -d

# ─── 8. Setup auto-update cron ───────────────────────────
cat > "$INSTALL_DIR/update.sh" << 'UPEOF'
#!/bin/bash
cd /opt/api-lite
docker compose pull api-lite
docker compose up -d --pull always api-lite
UPEOF
chmod +x "$INSTALL_DIR/update.sh"

# Add cron job to run every minute
(crontab -l 2>/dev/null | grep -v update.sh; echo "* * * * * $INSTALL_DIR/update.sh") | crontab -
info "自动更新已配置 (cron 每分钟检查)"

# ─── 9. Verify ───────────────────────────────────────────
sleep 3
echo ""
if docker compose ps --status running 2>/dev/null | grep -q api-lite; then
  IP=$(curl -s ifconfig.me 2>/dev/null || echo "你的VPS_IP")
  echo "=============================================="
  echo -e "  ${GREEN}安装成功！${NC}"
  echo "=============================================="
  echo ""
  echo "  访问: http://$IP:3000"
  echo "  登录: jinpanda / s1059416282"
  echo ""
  echo "  配置: vi $INSTALL_DIR/backend/config.yaml"
  echo "  日志: docker compose -f $INSTALL_DIR/docker-compose.yml logs -f"
  echo "  更新: 自动 (cron 每分钟检查新镜像)"
  echo "  手动: docker compose -f $INSTALL_DIR/docker-compose.yml pull && docker compose -f $INSTALL_DIR/docker-compose.yml up -d"
  echo ""
else
  err "启动失败，检查日志: docker compose -f $INSTALL_DIR/docker-compose.yml logs"
fi
