#!/bin/bash
set -e

REPO_URL="https://github.com/jinpanda11/api-lite.git"
INSTALL_DIR="/opt/api-lite"
SERVICE_NAME="api-lite"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()   { echo -e "${RED}[ERROR]${NC} $1"; }

echo "=============================================="
echo "  new-api-lite 一键安装"
echo "=============================================="
echo ""

if [ "$EUID" -ne 0 ]; then
  err "请以 root 运行: sudo bash install.sh"
  exit 1
fi

# Wait for apt lock (handles unattended-upgrades)
wait_for_apt() {
  local n=30
  while fuser /var/lib/dpkg/lock-frontend /var/lib/apt/lists/lock >/dev/null 2>&1; do
    n=$((n - 1))
    if [ $n -le 0 ]; then
      err "apt 被占用，请稍后重试"
      exit 1
    fi
    sleep 2
  done
}

# ─── 1. System ──────────────────────────────────────────
info "系统: $(uname -a | awk '{print $2, $NF}')"

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  aarch64) GOARCH="arm64" ;;
  *)       err "不支持的架构: $ARCH"; exit 1 ;;
esac

# ─── 2. Dependencies ────────────────────────────────────
if ! command -v git &>/dev/null; then
  info "安装 git..."
  wait_for_apt
  apt-get update -qq && apt-get install -y -qq git
fi

if ! command -v go &>/dev/null; then
  info "安装 Go..."
  wget -q "https://go.dev/dl/go1.23.0.linux-${GOARCH}.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  rm /tmp/go.tar.gz
  info "$(go version)"
else
  info "Go 已装: $(go version)"
fi

if ! command -v node &>/dev/null; then
  info "安装 Node.js..."
  wait_for_apt
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  wait_for_apt
  apt-get install -y -qq nodejs
  info "Node.js $(node -v) npm $(npm -v)"
else
  info "Node.js 已装: $(node -v)"
fi

# ─── 3. Clone ───────────────────────────────────────────
info "下载代码..."
if [ -d "$INSTALL_DIR" ]; then
  cd "$INSTALL_DIR"
  git pull
else
  git clone "$REPO_URL" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

# ─── 4. Config ──────────────────────────────────────────
if [ ! -f backend/config.yaml ]; then
  cp backend/config.yaml.example backend/config.yaml
  info "已创建 config.yaml（请修改 JWT secret 等参数）"
else
  info "config.yaml 已存在"
fi

# ─── 5. Build ───────────────────────────────────────────
info "构建前端..."
cd frontend
NODE_OPTIONS="--max-old-space-size=512" npm install --silent
NODE_OPTIONS="--max-old-space-size=512" npm run build
cd ..

info "构建后端..."
cd backend
go build -ldflags="-s -w" -o new-api-lite .
cd ..

# ─── 6. Service ─────────────────────────────────────────
info "注册系统服务..."
cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=api-lite
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/backend/new-api-lite
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl start $SERVICE_NAME

# ─── 7. Verify ─────────────────────────────────────────
sleep 2
echo ""
if systemctl is-active --quiet $SERVICE_NAME; then
  IP=$(curl -s ifconfig.me 2>/dev/null || echo "你的VPS_IP")
  echo "=============================================="
  echo -e "  ${GREEN}安装成功！${NC}"
  echo "=============================================="
  echo ""
  echo "  访问: http://$IP:3000"
  echo "  登录: jinpanda / s1059416282"
  echo ""
  echo "  配置: vi $INSTALL_DIR/backend/config.yaml"
  echo "  日志: journalctl -u $SERVICE_NAME -f"
  echo "  更新: cd $INSTALL_DIR && git pull && cd backend && go build && systemctl restart $SERVICE_NAME"
  echo ""
else
  err "启动失败，检查日志: journalctl -u $SERVICE_NAME -n 50"
fi
