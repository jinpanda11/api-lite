#!/bin/bash
set -e

REPO_URL="https://github.com/jinpanda11/api-lite.git"
INSTALL_DIR="/opt/new-api-lite"
SERVICE_NAME="new-api-lite"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()   { echo -e "${RED}[ERROR]${NC} $1"; }

echo "=============================================="
echo "  new-api-lite 一键安装脚本"
echo "=============================================="
echo ""

# ─── 0. Root check ─────────────────────────────────────────
if [ "$EUID" -ne 0 ]; then
  err "请以 root 用户运行: sudo bash install.sh"
  exit 1
fi

# ─── 1. 检测系统 ──────────────────────────────────────────
info "检测系统..."

OS=""
if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS=$ID
fi
if [[ "$OS" != "ubuntu" && "$OS" != "debian" ]]; then
  warn "仅测试过 Ubuntu/Debian，其他系统请自行安装依赖"
fi

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)       err "不支持的架构: $ARCH"; exit 1 ;;
esac
info "系统: $OS $(uname -m)"

# ─── 2. 安装依赖 ──────────────────────────────────────────
info "检查依赖..."

install_golang() {
  local ver=$(curl -s https://go.dev/VERSION?m=text | head -1)
  info "安装 Go ${ver} (${ARCH})..."
  wget -q "https://go.dev/dl/${ver}.linux-${ARCH}.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  rm /tmp/go.tar.gz
  info "Go 安装完成: $(go version)"
}

install_node() {
  info "安装 Node.js 20.x..."
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y nodejs
  info "Node.js 安装完成: $(node -v) npm: $(npm -v)"
}

# Git
if ! command -v git &>/dev/null; then
  apt-get update && apt-get install -y git
fi

# Go
if ! command -v go &>/dev/null; then
  install_golang
else
  info "Go 已安装: $(go version)"
fi

# Node.js + npm
if ! command -v node &>/dev/null; then
  install_node
else
  info "Node.js 已安装: $(node -v)"
fi

# ─── 3. 克隆代码 ──────────────────────────────────────────
info "克隆代码..."
if [ -d "$INSTALL_DIR" ]; then
  warn "$INSTALL_DIR 已存在，拉取最新代码..."
  cd "$INSTALL_DIR"
  git pull
else
  git clone "$REPO_URL" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

# ─── 4. 配置 ──────────────────────────────────────────────
info "配置..."
if [ -f backend/config.yaml ]; then
  warn "config.yaml 已存在，跳过创建"
else
  cp backend/config.yaml.example backend/config.yaml
  warn "已从模板创建 config.yaml，请编辑修改 JWT secret、SMTP 等参数"
  warn "  vi $INSTALL_DIR/backend/config.yaml"
fi

# ─── 5. 构建 ──────────────────────────────────────────────
info "构建前端..."
cd frontend
npm install --production=false
npm run build
cd ..

info "构建后端..."
cd backend
go build -ldflags="-s -w" -o new-api-lite .
cd ..

# ─── 6. 创建系统用户 ──────────────────────────────────────
if ! id -u deploy &>/dev/null; then
  info "创建 deploy 系统用户..."
  useradd -r -s /usr/sbin/nologin -d "$INSTALL_DIR" deploy
fi
chown -R deploy:deploy "$INSTALL_DIR"

# ─── 7. 注册 systemd 服务 ─────────────────────────────────
info "注册 systemd 服务..."
cat > /etc/systemd/system/$SERVICE_NAME.service << 'EOF'
[Unit]
Description=new-api-lite - API Management System
After=network.target

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/new-api-lite
ExecStart=/opt/new-api-lite/backend/new-api-lite
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
ProtectSystem=full
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl start $SERVICE_NAME

# ─── 8. 检查状态 ──────────────────────────────────────────
sleep 2
if systemctl is-active --quiet $SERVICE_NAME; then
  echo ""
  echo "=============================================="
  echo -e "  ${GREEN}✅ 安装成功！${NC}"
  echo "=============================================="
  echo ""
  echo "  访问地址: http://$(curl -s ifconfig.me 2>/dev/null || echo 'your-vps-ip'):3000"
  echo "  管理后台: http://your-vps-ip:3000"
  echo ""
  echo "  配置文件: $INSTALL_DIR/backend/config.yaml"
  echo "  查看日志: journalctl -u $SERVICE_NAME -f"
  echo "  重启服务: systemctl restart $SERVICE_NAME"
  echo "  更新代码: cd $INSTALL_DIR && sudo ./deploy.sh"
  echo ""
  echo "  ⚠️  重要：请务必修改 config.yaml 中的以下配置："
  echo "     1. jwt.secret     → 改为随机字符串"
  echo "     2. smtp           → 配置 SMTP 信息"
  echo "     3. admin.password → 管理员密码"
  echo ""
else
  err "服务启动失败，检查日志: journalctl -u $SERVICE_NAME -n 50"
fi
