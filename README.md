# FluxPanel

一个综合性的监控与 AI 助手平台，集成了客户端监控、通知推送、iLink AI 助手等功能。

## 功能特性

### 客户端监控
- 实时数据上报与可视化展示
- WebSocket 实时推送
- 客户端状态监控（心跳、在线状态）
- 自定义排序与分组
- 告警阈值管理

### 通知系统
- 多渠道通知支持（iLink、飞书）
- 灵活的通知规则配置
- 天气预报定时推送
- 告警阈值触发通知

### iLink AI 助手
- 普通聊天与上下文记忆
- Todo 待办事项管理
- 定时提醒功能
- 可扩展技能系统
- 多模型支持（OpenAI、通义千问等）

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go + Gin + GORM |
| 前端 | React + TypeScript + Vite + Tailwind CSS |
| 数据库 | PostgreSQL |
| 实时通信 | WebSocket |
| 图表 | Recharts |
| 部署 | Docker Compose |

## 快速开始

### 使用 Docker Compose（推荐）

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

访问 http://localhost:3000 查看监控面板。

### 本地开发

#### 后端

```bash
cd backend

# 安装依赖
go mod tidy

# 启动服务（需要先启动 PostgreSQL）
go run main.go
```

#### 前端

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

## API 接口

### 监控相关

#### POST /api/report
客户端上报数据

```bash
curl -X POST http://localhost:8080/api/report \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "event_type": "heartbeat",
    "data": {"cpu": 50, "memory": 60},
    "status": "success"
  }'
```

#### GET /api/summary
获取汇总统计数据

#### GET /api/events
获取事件列表（支持分页和筛选）

参数：
- `page`: 页码
- `page_size`: 每页数量
- `client_id`: 客户端ID筛选
- `status`: 状态筛选
- `event_type`: 事件类型筛选

#### GET /api/stats/hourly
获取今日每小时事件统计

#### GET /api/stats/clients
获取客户端事件排名

#### WebSocket /ws
实时推送新事件

### AI 助手相关

#### GET/PUT /api/assistant/llm
LLM 配置管理

#### GET/POST/PUT/DELETE /api/assistant/todos
Todo 待办事项管理

#### GET/DELETE /api/assistant/memories
记忆管理

#### GET /api/assistant/reminders
提醒列表

### 通知管理

#### GET/POST/PUT/DELETE /api/notifications/channels
通知渠道管理

#### GET/POST/PUT/DELETE /api/notifications/rules
通知规则管理

#### GET/POST/PUT/DELETE /api/alerts/thresholds
告警阈值管理

### iLink 登录

#### GET /api/wecom/login/qrcode
获取 iLink 登录二维码

#### GET /api/wecom/login/status
获取登录状态

#### DELETE /api/wecom/session
退出登录

## 数据结构

### 事件 (Event)
```typescript
interface Event {
  id: number
  client_id: string      // 客户端唯一标识
  event_type: string     // 事件类型
  data: object           // 自定义数据
  status: string         // success | error | warning
  created_at: string     // 时间戳
}
```

### AI 助手用户 (AIUser)
```typescript
interface AIUser {
  id: number
  wecom_user_id: string  // 微信用户ID
  name: string           // 用户名称
  created_at: string
}
```

### Todo
```typescript
interface Todo {
  id: number
  user_id: number
  content: string        // Todo内容
  deadline?: string      // 截止时间
  completed: boolean     // 是否完成
  created_at: string
}
```

### 提醒 (Reminder)
```typescript
interface Reminder {
  id: number
  user_id: number
  content: string        // 提醒内容
  remind_at: string      // 提醒时间
  sent: boolean          // 是否已发送
  created_at: string
}
```

## 环境变量

### 后端

| 变量 | 默认值 | 说明 |
|------|--------|------|
| SERVER_PORT | 8080 | 服务端口 |
| DB_HOST | localhost | 数据库地址 |
| DB_PORT | 5432 | 数据库端口 |
| DB_USER | postgres | 数据库用户 |
| DB_PASSWORD | postgres | 数据库密码 |
| DB_NAME | client_monitor | 数据库名称 |

### AI 助手配置（通过管理界面配置）

| 变量 | 说明 |
|------|------|
| LLM_PROVIDER | LLM 提供商 (openai, qwen) |
| LLM_API_KEY | API Key |
| LLM_MODEL | 模型名称 |
| LLM_BASE_URL | API Base URL（可选） |

## 项目结构

```
FluxPanel/
├── backend/                 # 后端服务
│   ├── main.go             # 入口文件
│   ├── config/             # 配置管理
│   ├── database/           # 数据库连接
│   ├── models/             # 数据模型
│   ├── handlers/           # HTTP 处理器
│   ├── services/           # 业务服务
│   ├── agent/              # Agent 适配器
│   ├── skill/              # 技能系统
│   ├── notify/             # 通知服务
│   ├── ilink/              # iLink 协议实现
│   └── messaging/          # 消息处理
├── frontend/                # 前端应用
│   ├── src/
│   │   ├── components/     # React 组件
│   │   ├── hooks/          # React Hooks
│   │   ├── services/       # 前端服务
│   │   └── types/          # TypeScript 类型
│   └── package.json
├── docker-compose.yml       # Docker 编排
├── CLAUDE.md               # 详细项目文档
└── README.md               # 项目简介（本文件）
```

## 许可证

MIT
