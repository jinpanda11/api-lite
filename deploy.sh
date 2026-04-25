#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_NAME="new-api-lite"

echo "=============================================="
echo "  new-api-lite 部署脚本"
echo "=============================================="

cd "$PROJECT_DIR"

# 1. 拉取最新代码
echo "[1/4] 拉取最新代码..."
git pull origin main

# 2. 构建前端
echo "[2/4] 构建前端..."
cd frontend
npm install --production=false
npm run build
cd ..

# 3. 构建后端（嵌入前端静态文件）
echo "[3/4] 构建后端..."
cd backend
go build -ldflags="-s -w" -o new-api-lite .
cd ..

# 4. 复制配置文件（如果不存在）
if [ ! -f backend/config.yaml ]; then
    echo "[INFO] backend/config.yaml 不存在，从模板创建..."
    cp backend/config.yaml.example backend/config.yaml
    echo "[WARN] 请编辑 backend/config.yaml 配置你的实际参数（JWT密钥、SMTP等）"
fi

# 5. 重启服务
echo "[4/4] 重启服务..."
if systemctl is-active --quiet "$SERVICE_NAME"; then
    sudo systemctl daemon-reload
    sudo systemctl restart "$SERVICE_NAME"
    echo "服务已重启"
else
    echo "提示: $SERVICE_NAME 服务未运行。"
    echo "你可以手动启动: sudo systemctl start $SERVICE_NAME"
    echo "或设置开机自启: sudo systemctl enable $SERVICE_NAME"
fi

echo "=============================================="
echo "  部署完成！"
echo "=============================================="
