# New API Lite

一个轻量化的 OpenAI 兼容 API 中转系统，基于 [new-api](https://github.com/QuantumNous/new-api) 精简实现。

## 功能

- **纯转发中继**：将 `/v1/*` 请求透明代理到上游渠道，支持 stream 流式返回
- **用户系统**：注册（邮箱验证码）、登录、JWT 认证
- **令牌管理**：创建 `sk-xxx` 格式 API Key，设置过期时间、备注
- **渠道管理**：配置多个上游渠道，按优先级自动选择，按模型过滤
- **计费系统**：按输入/输出 tokens 计费，自动扣减用户余额
- **兑换码充值**：管理员生成兑换码，用户凭码充值
- **使用日志**：记录每次请求的模型、tokens、费用、状态
- **Dashboard**：余额卡片、7 天趋势图（Chart.js）

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22 + Gin + GORM v2 + JWT + Viper |
| 数据库 | SQLite（开发）/ MySQL（生产） |
| 前端 | React 18 + TypeScript + Vite + Semi Design |
| 状态 | Zustand（持久化 token、主题） |

---

## 快速开始

### 后端

```bash
cd backend

# 安装依赖
go mod tidy

# 复制并编辑配置（SMTP 可选，调试模式下验证码打印到控制台）
cp config.yaml config.yaml   # 已有默认配置

# 启动（默认端口 3000，SQLite 数据库 new-api-lite.db）
go run .
```

首次启动自动创建管理员账号（见 `config.yaml` → `admin` 节）。

### 前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式（代理 /api 和 /v1 到 localhost:3000）
npm run dev

# 生产构建（产物输出到 backend/web/）
npm run build
```

### 生产部署（单文件）

前端 build 后，Go 服务可以托管静态文件：

```bash
# 在 backend/main.go 已有的路由下追加（可选）：
# r.Static("/", "./web")
```

---

## 目录结构

```
new-api-lite/
├── backend/
│   ├── main.go
│   ├── config.yaml          # 配置文件
│   ├── config/config.go
│   ├── model/               # GORM 模型 + 数据库操作
│   ├── middleware/          # JWT 认证、CORS
│   ├── handler/             # HTTP 处理器
│   │   ├── relay.go         # 核心：纯转发中继
│   │   ├── user.go          # 注册、登录、验证码
│   │   ├── token.go         # 令牌 CRUD
│   │   ├── channel.go       # 渠道 + 模型列表
│   │   ├── log.go           # 使用记录、Dashboard
│   │   ├── wallet.go        # 余额、兑换码
│   │   └── admin.go         # 管理员：兑换码生成、用户管理
│   ├── service/email.go     # SMTP 邮件发送
│   └── router/router.go     # 路由注册
└── frontend/
    └── src/
        ├── api/index.ts     # axios 封装
        ├── store/index.ts   # Zustand 全局状态
        ├── components/Layout.tsx
        └── pages/           # 各功能页面
```

---

## API 接口概览

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/verification?email=xxx` | 发送邮箱验证码 |
| POST | `/api/user/email/code` | 发送邮箱验证码（备用路径） |
| POST | `/api/user/register` | 注册 |
| POST | `/api/user/login` | 登录，返回 JWT |

### 认证接口（Bearer JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/user/info` | 当前用户信息 |
| POST | `/api/user/update-password` | 修改密码 |
| GET | `/api/dashboard` | 统计数据 + 7天趋势 |
| GET/POST/PUT/DELETE | `/api/token[/:id]` | 令牌 CRUD |
| GET | `/api/models` | 可用模型列表 |
| GET | `/api/log` | 使用记录（分页） |
| GET | `/api/balance` | 当前余额 |
| POST | `/api/redeem` | 兑换码充值 |
| GET | `/api/topup/logs` | 充值记录 |

### 管理员接口（Bearer JWT + role=admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/channel[/:id]` | 渠道 CRUD |
| GET/POST/DELETE | `/api/admin/redeem[/:id]` | 兑换码管理 |
| GET/PUT | `/api/admin/user[/:id]` | 用户列表/编辑 |

### 中继接口（Bearer sk-xxx）

```
ANY /v1/*  →  透明转发到匹配渠道
```

---

## 配置说明

```yaml
server:
  port: 3000
  debug: false        # true 时输出 SQL 日志

database:
  driver: sqlite      # sqlite | mysql
  dsn: "new-api-lite.db"

jwt:
  secret: "your-secret-key"
  expire_hours: 168   # 7天

smtp:
  host: "smtp.gmail.com"
  port: 465
  username: "you@gmail.com"
  password: "app-password"
  from: "New API Lite <you@gmail.com>"
  ssl: true
  # 若 host 留空或为 smtp.example.com，验证码打印到控制台（开发模式）

admin:
  username: admin
  email: admin@example.com
  password: Admin123456   # 首次启动时自动创建
```

---

## License

AGPLv3（参考 new-api 上游协议）
