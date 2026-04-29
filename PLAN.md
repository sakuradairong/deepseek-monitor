# 完整实现计划 — 实时监控 + 请求代理 + 日志系统

## 架构变更

### 新增核心组件

1. **反向代理层** (`api/proxy.go`)
   - 接受 OpenAI 兼容的请求 (`POST /v1/chat/completions`)
   - 轮转多 API Key 转发到 DeepSeek
   - 记录每次调用的延迟、状态码、Token 用量

2. **实时指标追踪器** (`api/metrics.go`)
   - 内存中的滑动窗口 QPS 计数器
   - 实时延迟/错误率/cost 聚合
   - 每分钟一个数据点供图表展示

3. **API 调用日志** (文件 + 数据库)
   - 数据库: `proxy_logs` 表 (持久化历史)
   - 文件: `logs/api_calls.log` (JSON 行格式, 按日期轮转)

### 新增数据模型
- `ProxyLog`: id, model, api_model, prompt_tokens, completion_tokens, total_tokens, latency_ms, status_code, error_type, prompt_preview, response_preview, api_key_id, created_at

### 新增 API 端点
- `POST /v1/chat/completions` — 代理端点 (OpenAI 兼容, 带 Key 轮转)
- `GET /api/v1/proxy/logs` — 分页查询调用日志
- `GET /api/v1/proxy/realtime` — 实时指标 (QPS/延迟/错误率)

### 前端新增视图
- **实时监控页** — QPS 实时曲线, 延迟热力图, 错误率仪表
- **调用日志页** — 分页表格, 按时间/状态/模型过滤
- **仪表盘增强** — 增加 QPS / 延迟 / 错误率卡片
