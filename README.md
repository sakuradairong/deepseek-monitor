# DeepSeek API Monitor

全功能 DeepSeek API 监控 + 反向代理平台。集**使用量监控、实时 QPS/延迟追踪、费用统计、多 API Key 自动轮转、账户系统**于一体。

## 功能特性

### 📊 监控仪表盘
- **余额监控** — 实时跟踪账户余额变化历史，低于阈值自动告警
- **Token 用量分析** — 按模型和时间维度展示 token 消耗趋势
- **费用统计** — 日/月费用统计，各模型费用占比
- **速率限制监控** — 跟踪 API 速率限制剩余额度
- **错误追踪** — 记录并展示 API 调用异常

### ⚡ 反向代理层
- **透明代理** — `POST /v1/chat/completions` 100% OpenAI 兼容，无缝替换 API 地址
- **多 Key 自动轮转** — 按优先级 + 使用次数自动选择最优 Key
- **Key 在线测试** — 添加后立即验证有效性

### 📈 实时监控
- **QPS 实时曲线** — 60 秒滑动窗口，每秒数据点，2 秒自动刷新
- **延迟追踪** — 平均延迟 / P95 / P99
- **错误率监控** — 4xx / 5xx 分类统计
- **Token 吞吐量** — tokens/sec 实时曲线
- **费用实时统计** — 窗口内 cost 实时展示

### 📄 调用日志
- **DB 持久化** — 每次代理调用的详细记录
- **文件日志** — `logs/api_calls.log` 结构化 JSON 格式
- **Prompt/响应预览** — 记录输入质量和输出 token 数
- **日志查询** — 支持按模型/状态码/错误类型过滤

### 🔐 账户系统
- **JWT 认证** — 首次注册自动管理员
- **多用户支持** — 管理员可创建子账号
- **鉴权隔离** — 敏感 API 需认证访问

### ⚙️ 系统管理
- **前端设置界面** — 采集间隔、数据保留、告警阈值在线调
- **数据自动清理** — 超过配置天数的数据自动清理

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.22+ (Gin + GORM) |
| 前端 | Vue3 + Vite + Element Plus + ECharts |
| 数据库 | SQLite (可迁移至 PostgreSQL) |
| 认证 | JWT + bcrypt |
| 定时器 | robfig/cron |
| 代理 | 原生 http.Client + Key 轮转 |

## 快速开始

### 1. 前置要求

- Go 1.22+
- Node.js 18+ (仅首次构建需要)
- DeepSeek API Key

### 2. 一键启动

```bash
./start.sh
# 浏览器打开 http://localhost:8080
```

启动脚本会自动：Go 编译 → 前端构建 → 启动服务。

### 3. 首次使用

1. 打开 `http://localhost:8080`，会看到登录页面
2. 输入任意用户名/密码（如 `admin` / `admin123`），系统会自动注册为管理员
3. 登录后进入 **API Keys** 页面，添加你的 DeepSeek API Key
4. 点击 **测试** 验证 Key 有效性
5. 返回 **仪表盘** 查看数据

### 4. 集成到你的应用

将你的 AI 应用 API 地址改为本监控服务：

```bash
# 原来 — 直连 DeepSeek
OPENAI_BASE_URL=https://api.deepseek.com

# 现在 — 经过监控代理（自带 Key 轮转）
OPENAI_BASE_URL=http://your-server:8080
```

所有 API 调用自动：轮转 Key、记录延迟状态 Token、更新实时 QPS 指标、写入日志文件。

### 5. 开发模式

```bash
# 终端1: 启动后端
cd backend && go run . config.yaml

# 终端2: 启动前端开发服务器（支持热重载 + API 代理）
cd frontend && npm run dev
# 访问 http://localhost:5173
```

## 访问方式

| 模式 | 地址 | 说明 |
|------|------|------|
| 生产模式 | `http://localhost:8080` | 单端口，前后端一体 |
| 开发模式 | `http://localhost:5173` | Vite HMR，API 自动代理到 8080 |

## API 接口

### 公开接口

| 端点 | 说明 |
|------|------|
| `GET /api/v1/health` | 健康检查 |
| `POST /api/v1/auth/login` | 用户登录 |
| `POST /api/v1/auth/register` | 用户注册（首次自动管理员） |
| `POST /v1/chat/completions` | **代理端点** — OpenAI 兼容，自动 Key 轮转 |

### 认证接口（需 `Authorization: Bearer <token>`）

| 端点 | 说明 |
|------|------|
| `GET /api/v1/auth/me` | 当前用户信息 |

### 数据查询

| 端点 | 说明 |
|------|------|
| `GET /api/v1/stats/overview` | 仪表盘总览 |
| `GET /api/v1/stats/balance` | 最新余额 |
| `GET /api/v1/stats/balance/history?days=30` | 余额历史 |
| `GET /api/v1/stats/usage/trend?days=7` | 用量趋势 |
| `GET /api/v1/stats/usage/summary?days=30` | 用量汇总 |
| `GET /api/v1/stats/usage/models?days=30` | 模型分布 |
| `GET /api/v1/stats/ratelimit` | 速率限制 |
| `GET /api/v1/stats/errors?limit=20` | 错误记录 |

### 代理日志与实时指标

| 端点 | 说明 |
|------|------|
| `GET /api/v1/proxy/logs?offset=0&limit=50&model=&error_type=&min_status=` | 代理调用日志（分页+过滤） |
| `GET /api/v1/proxy/realtime` | 实时指标（QPS/延迟/错误率/费用） |

### 系统管理

| 端点 | 说明 |
|------|------|
| `GET /api/v1/settings` | 读取系统设置 |
| `PUT /api/v1/settings` | 更新系统设置 |
| `GET /api/v1/keys` | 列出全部 API Key |
| `POST /api/v1/keys` | 添加 API Key |
| `PUT /api/v1/keys/:id` | 更新 API Key |
| `DELETE /api/v1/keys/:id` | 删除 API Key |
| `POST /api/v1/keys/:id/test` | 测试 API Key 有效性 |
| `GET /api/v1/keys/names` | 获取 Key 名称列表（下拉选择用） |

## 多 API Key 轮转

系统自动管理多个 API Key 的轮转使用：

- **优先级驱动** — `priority` 越高的 Key 优先使用
- **负载均衡** — 同等优先级下，使用次数最少的 Key 被选中
- **自动故障转移** — Key 出错时自动切换到下一个可用 Key
- **状态追踪** — 每个 Key 记录最后使用时间、累计调用次数、最近错误

在 **API Keys** 页面管理 Key，支持在线测试。

## 实时指标

- **滑动窗口** — 最近 60 秒的数据点
- **QPS** — 每秒请求数实时曲线
- **延迟** — 平均/P95/P99 延迟
- **错误率** — 4xx/5xx 百分比
- **Token 吞吐** — 每秒处理 token 数
- **窗口费用** — 60 秒内的估算费用

前端每 2 秒自动刷新实时数据。

## 调用日志

每次经由代理的 API 调用都会记录：

- **时间戳** — 精确到微秒
- **模型名** — 请求模型和实际响应模型
- **HTTP 状态码** — 200/4xx/5xx
- **延迟** — 毫秒级延迟
- **Token 用量** — Prompt / Completion / Total
- **错误类型** — 4xx / 5xx / network_error
- **Prompt 预览** — 输入消息前 500 字符
- **响应预览** — 输出内容前 500 字符
- **使用 Key** — 实际调用了哪个 Key

日志同时写入数据库（可查询）和文件（`logs/api_calls.log`，JSON 行格式）。

## 配置说明

编辑 `backend/config.yaml`:

```yaml
server:
  port: 8080
  host: 0.0.0.0

database:
  driver: sqlite          # sqlite 或 postgres
  dsn: data/deepseek_monitor.db

deepseek:
  base_url: https://api.deepseek.com
  api_key: ""             # 可选 — Key 从 Web 界面管理

monitor:
  collect_interval: 5m    # 定时采集合集间隔
  retention_days: 90      # 数据保留天数

log:
  level: info
  format: text
```

设置也可通过前端 **系统设置** 页面在线修改。

## 迁移到 PostgreSQL

1. 安装 PostgreSQL 并创建数据库
2. 修改 `config.yaml`:
   ```yaml
   database:
     driver: postgres
     dsn: "host=localhost user=postgres password=xxx dbname=deepseek_monitor port=5432 sslmode=disable"
   ```
3. 在 `database/db.go` 中添加 PostgreSQL 驱动导入
4. 重启服务（GORM AutoMigrate 自动迁移表结构）

## 项目结构

```
deepseek-monitor/
├── start.sh                      # 一键启动脚本
├── docker-compose.yml            # Docker 部署
├── README.md
│
├── backend/
│   ├── main.go                   # 入口 (初始化 DB/调度/服务)
│   ├── config.yaml               # 配置文件
│   ├── config/config.go          # 配置结构 + 加载
│   │
│   ├── models/models.go          # 数据模型 (9 张表)
│   │   ├── User                  # 账户系统
│   │   ├── APIKey                # 多 Key 管理
│   │   ├── SystemConfig          # 系统设置 KV
│   │   ├── ProxyLog              # 代理调用日志
│   │   ├── BalanceSnapshot       # 余额快照
│   │   ├── UsageRecord           # 用量记录
│   │   ├── RateLimitRecord       # 速率限制
│   │   ├── DailyUsageSummary     # 日汇总
│   │   └── APIErrorRecord        # 错误记录
│   │
│   ├── database/
│   │   ├── db.go                 # 数据库初始化 + AutoMigrate
│   │   └── repository.go         # 数据访问层
│   │
│   ├── collector/
│   │   └── collector.go          # DeepSeek HTTP 客户端
│   │
│   ├── api/
│   │   ├── auth.go               # JWT 生成/验证/中间件
│   │   ├── auth_handlers.go      # 登录/注册/用户
│   │   ├── handlers.go           # 仪表盘数据 API
│   │   ├── key_handlers.go       # API Key CRUD + 测试
│   │   ├── settings_handlers.go  # 系统设置 API
│   │   ├── proxy.go              # 反向代理 + 调用记录
│   │   ├── metrics.go            # 实时 QPS/延迟/错误率追踪
│   │   ├── middleware.go         # 日志中间件
│   │   └── router.go             # 路由表
│   │
│   ├── scheduler/
│   │   └── scheduler.go          # 定时采集 + 多 Key 轮转
│   │
│   ├── web/dist/                 # 前端构建产物 (由 build 生成)
│   └── logs/api_calls.log        # API 调用日志文件
│
└── frontend/
    └── src/
        ├── App.vue               # 根组件 (登录页 + 侧边栏布局)
        ├── main.js               # Vue 入口
        ├── store/auth.js         # 认证状态管理
        ├── api/index.js          # Axios 客户端 (含 Token 拦截器)
        └── views/
            ├── Dashboard.vue     # 监控仪表盘
            ├── Realtime.vue      # 实时 QPS/延迟监控
            ├── Logs.vue          # 调用日志查询
            ├── Keys.vue          # API Key 管理
            └── Settings.vue      # 系统设置 + 用户管理
```

## 使用流程

```
┌─────────────────────────────────────────────────────────────┐
│                    你的 AI 应用                               │
│       OpenAI SDK 指向 http://your-server:8080               │
└─────────────────────┬───────────────────────────────────────┘
                      │ POST /v1/chat/completions
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                DeepSeek API Monitor                          │
│                                                             │
│  1. 选择最优 API Key (轮转)                                   │
│  2. 转发请求到 api.deepseek.com                              │
│  3. 记录: 延迟 / 状态码 / Token 用量                         │
│  4. 更新实时 QPS 指标                                         │
│  5. 写入日志 (DB + 文件)                                      │
│  6. 更新 Key 使用计数                                         │
│                                                             │
│  ┌──────────────────────────────────────────────────┐       │
│  │  定时采集器 (5m 间隔)                              │       │
│  │  → GET /user/balance (余额)                       │       │
│  │  → POST /chat/completions (用量探测)               │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────────┘
                      │ 原始响应透传
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Vue3 仪表盘                                │
│  仪表盘 │ 实时监控 │ 调用日志 │ API Keys │ 系统设置           │
└─────────────────────────────────────────────────────────────┘
```

## DeepSeek API 定价（官方人民币报价）

| 模型 | 缓存命中输入 (元/1M tokens) | 缓存未命中输入 (元/1M) | 输出 (元/1M) |
|------|---------------------------|----------------------|-------------|
| deepseek-v4-flash | ¥0.02 | ¥1 | ¥2 |
| deepseek-v4-pro | ¥0.025 (2.5折) | ¥3 | ¥6 |

> Note: 以上为人民币 (CNY/元) 报价。监控系统内的「费用」字段以元为单位。
> USD 参考：按 7.2 汇率，v4-flash 输出约 $0.28/1M tokens。

## Docker 部署

```bash
# 1. 构建
cd backend && docker build -t deepseek-monitor .

# 2. 运行
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  deepseek-monitor

# 3. 或用 docker-compose
docker-compose up -d
```

## License

MIT
