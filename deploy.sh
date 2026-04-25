#!/bin/bash
set -e

cd /opt/api-lite

echo "=== 更新 api-lite ==="

git pull

echo "构建前端..."
cd frontend && NODE_OPTIONS="--max-old-space-size=512" npm install --silent && NODE_OPTIONS="--max-old-space-size=512" npm run build && cd ..

echo "构建后端..."
cd backend && go build -ldflags="-s -w" -o new-api-lite . && cd ..

echo "重启服务..."
systemctl restart api-lite

echo "完成！"
