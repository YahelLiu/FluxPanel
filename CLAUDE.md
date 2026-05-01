# FluxPanel

## 项目概述

FluxPanel 是一个综合性的监控与 AI 助手平台，集成了客户端监控、通知推送、iLink AI 助手等功能。系统采用前后端分离架构，后端使用 Go 语言开发，前端使用 React + TypeScript。

---

## 核心模块

### 1. 客户端监控面板
- 实时数据上报与可视化展示
- 客户端状态监控（心跳、在线状态）
- 自定义排序与分组
- WebSocket 实时推送

### 2. 通知系统
- 多渠道通知支持（iLink、飞书等）
- 灵活的通知规则配置
- 天气预报定时推送
- 告警阈值管理

### 3. iLink AI 助手
- 普通聊天与上下文记忆
- Todo 待办事项管理
- 定时提醒功能
- 可扩展技能系统

---

# iLink AI Assistant 模块

## 概述

iLink AI 助手模块，集成到 FluxPanel 中，通过 iLink 协议实现微信消息收发，共享现有的通知系统、用户体系和定时任务基础设施。

---

## AI 助手核心功能

### 1. 普通聊天
- 接收用户消息，结合上下文和记忆，调用 LLM 生成回复
- 支持多轮对话，保持对话上下文

### 2. 记忆管理
- 用户可以告诉 AI 记住特定信息
- AI 在后续对话中会参考这些记忆
- 示例：
  - "记住我喜欢简洁一点的回答"
  - "记住我的项目叫 X"
  - "以后叫我老李"

### 3. Todo 管理
- **创建 Todo**：用户可以添加待办事项，可选截止时间
- **查看 Todo**：列出所有未完成的待办事项
- **完成 Todo**：标记待办事项为已完成
- 示例：
  - "帮我加个 todo，明天整理合同"
  - "我有哪些 todo"
  - "完成整理合同这个 todo"

### 4. 提醒功能
- 创建一次性提醒
- 查看提醒列表
- 取消提醒
- 到时间主动发送微信消息
- 示例：
  - "30分钟后提醒我开会"
  - "明天上午10点提醒我发邮件"
  - "今晚8点叫我看文档"
  - "我有哪些提醒"
  - "取消开会的提醒"

---

## 系统架构

### 核心链路

```
用户在微信发送消息
        ↓
    iLink 收到消息
        ↓
   意图判断（Agent）
        ↓
   执行对应能力
        ↓
  结果回复到微信
```

### Agent 决策逻辑

每条消息先进入统一判断器，返回结构化结果：

```json
{
  "intent": "chat | memory | todo | reminder",
  "action": "create | list | complete | none",
  "content": "具体内容",
  "time": "相关时间（可选）"
}
```

---

## 数据模型

### AI 用户 (AIUser)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| wecom_user_id | string | 微信用户ID |
| name | string | 用户名称 |
| created_at | timestamp | 创建时间 |

### 对话记录 (Conversation)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 用户ID |
| role | string | user 或 assistant |
| content | text | 消息内容 |
| created_at | timestamp | 创建时间 |

### 记忆 (Memory)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 用户ID |
| content | text | 记忆内容 |
| created_at | timestamp | 创建时间 |

### Todo
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 用户ID |
| content | text | Todo内容 |
| deadline | timestamp | 截止时间（可选） |
| completed | bool | 是否完成 |
| created_at | timestamp | 创建时间 |

### 提醒 (Reminder)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 用户ID |
| content | text | 提醒内容 |
| remind_at | timestamp | 提醒时间 |
| sent | bool | 是否已发送 |
| created_at | timestamp | 创建时间 |

---

## 工作流详细设计

### 工作流 1：聊天
```
收到消息
    ↓
判断为普通聊天 (intent = chat)
    ↓
读取最近对话历史（最近10条）
读取用户相关记忆
    ↓
调用 LLM 生成回复
    ↓
保存对话记录
回复到微信
```

### 工作流 2：保存记忆
```
收到消息
    ↓
判断为记忆 (intent = memory, action = create)
    ↓
提取记忆内容
    ↓
保存到 Memory 表
    ↓
回复："好的，我记住了"
```

### 工作流 3：创建 Todo
```
收到消息
    ↓
判断为 Todo (intent = todo, action = create)
    ↓
提取 Todo 内容和截止时间
    ↓
保存到 Todo 表
    ↓
回复确认
```

### 工作流 4：查看 Todo
```
收到消息
    ↓
判断为查看 Todo (intent = todo, action = list)
    ↓
查询用户未完成的 Todo
    ↓
整理成列表格式
    ↓
回复列表
```

### 工作流 5：完成 Todo
```
收到消息
    ↓
判断为完成 Todo (intent = todo, action = complete)
    ↓
匹配对应的 Todo 项
    ↓
标记为已完成
    ↓
回复确认
```

### 工作流 6：创建提醒
```
收到消息
    ↓
判断为提醒 (intent = reminder, action = create)
    ↓
提取提醒内容和时间
    ↓
保存到 Reminder 表
    ↓
回复确认
```

### 工作流 7：发送提醒（定时任务）
```
定时检查（每分钟）
    ↓
查询 remind_at <= 当前时间 且 sent = false 的提醒
    ↓
主动发送微信消息
    ↓
标记 sent = true
```

---

## API 设计

### iLink 消息接口
通过 iLink 协议收发微信消息，无需企业微信应用配置

### AI 助手管理接口
```
GET  /api/assistant/todos       # 获取 Todo 列表
POST /api/assistant/todos       # 创建 Todo
PUT  /api/assistant/todos/:id   # 更新 Todo（完成）
GET  /api/assistant/memories    # 获取记忆列表
DELETE /api/assistant/memories/:id  # 删除记忆
GET  /api/assistant/reminders   # 获取提醒列表
```

---

## 与 FluxPanel 集成

### 复用现有模块

1. **iLink 通知渠道**
   - 复用 `notify/drivers/ilink.go` 中的 iLink 驱动
   - 通过 iLink 协议发送微信消息

2. **通知渠道配置**
   - 复用 `NotificationChannel` 模型
   - 复用 iLink 登录状态管理

3. **定时任务**
   - 复用 `notify/weather.go` 的调度模式
   - 复用 `notify/reminder.go` 处理提醒调度

4. **数据库**
   - 复用现有 GORM 和数据库连接
   - 在 `database/db.go` 中添加新模型迁移

### 文件结构

```
backend/
├── handlers/
│   ├── wecom.go          # iLink 消息处理
│   └── assistant.go      # AI 助手 API
├── models/
│   └── assistant.go      # AI 助手数据模型
├── services/
│   ├── agent.go          # 意图判断
│   └── llm.go            # LLM 调用
├── notify/
│   └── reminder.go       # 提醒调度
├── ilink/                # iLink 协议
│   ├── client.go
│   ├── auth.go
│   └── monitor.go
└── main.go               # 添加新路由
```

---

## 配置

AI 助手通过管理界面配置，存储在数据库中：

```
# LLM 配置
LLM_PROVIDER=openai
LLM_API_KEY=your-api-key
LLM_MODEL=gpt-4o-mini
LLM_BASE_URL=https://api.openai.com/v1  # 可选，用于其他兼容 API
```

iLink 渠道通过扫码登录自动获取凭证，无需手动配置。

---

## MVP 边界

### 第一版包含
- ✅ 单用户或少量用户
- ✅ 文字消息
- ✅ 简单记忆
- ✅ 简单 Todo
- ✅ 一次性提醒
- ✅ 最近对话上下文
- ✅ iLink 微信收发

### 第一版不包含
- ❌ 复杂权限系统
- ❌ 多人协作
- ❌ 循环提醒
- ❌ 复杂日历
- ❌ 网页后台管理
- ❌ 语音/图片消息
- ❌ 复杂知识库
- ❌ 插件市场

---

## 验收标准

以下场景全部跑通即为第一版完成：

| 序号 | 用户输入 | 期望 AI 回复 |
|------|----------|--------------|
| 1 | 你好 | 正常打招呼回复 |
| 2 | 记住我喜欢简洁回答 | "好的，我记住了" |
| 3 | 帮我加个 todo，明天整理方案 | "已添加：明天整理方案" |
| 4 | 我有哪些 todo | 列出未完成事项 |
| 5 | 1分钟后提醒我测试 | "好的，1分钟后提醒你" |
| 6 | （1分钟后） | AI 主动发消息："提醒你测试" |

---

## 实现优先级

### P0 - 基础框架（必须先完成）
1. 数据模型定义（`models/assistant.go`）
2. 数据库迁移
3. LLM 服务封装（`services/llm.go`）
4. Agent 意图判断（`services/agent.go`）

### P1 - iLink 集成
1. iLink 消息监听（`ilink/monitor.go`）
2. iLink 消息发送
3. 登录状态管理

### P2 - 核心功能
1. 聊天功能
2. 记忆管理
3. Todo 管理
4. 提醒创建

### P3 - 定时任务
1. 提醒调度器（`notify/reminder.go`）
2. 主动消息发送

---

## Prompt 设计

### 意图判断 Prompt

```
你是一个意图识别助手。分析用户消息，返回 JSON 格式的意图。

用户消息：{message}

返回格式：
{
  "intent": "chat|memory|todo|reminder",
  "action": "create|list|complete|none",
  "content": "提取的内容",
  "time": "时间描述（如有）"
}

判断规则：
- intent=memory: 用户想让你记住某事（"记住..."、"以后..."）
- intent=todo: 用户想管理待办事项（"加个todo"、"我有哪些todo"、"完成..."）
- intent=reminder: 用户想设置提醒（"X分钟后提醒我"、"明天X点提醒我"）
- intent=chat: 普通聊天

action 规则：
- todo: create（创建）、list（查看列表）、complete（完成）
- reminder: create（创建）
- memory: create（创建）
- chat: none

只返回 JSON，不要其他内容。
```

### 聊天回复 Prompt

```
你是一个友好的 AI 助手。

用户信息：
- 名称：{user_name}

用户记忆：
{memories}

最近对话：
{conversations}

用户消息：{message}

请根据上下文和记忆，友好地回复用户。如果用户记忆中有偏好，请遵守。
```

---

## 项目目录结构

```
FluxPanel/
├── backend/                    # 后端服务
│   ├── main.go                # 入口文件
│   ├── config/                # 配置管理
│   │   └── config.go
│   ├── database/              # 数据库连接
│   │   └── db.go
│   ├── models/                # 数据模型
│   │   ├── assistant.go       # AI 助手相关模型
│   │   ├── notification.go    # 通知相关模型
│   │   ├── alert.go           # 告警模型
│   │   ├── weather.go         # 天气模型
│   │   ├── event.go           # 事件模型
│   │   ├── client_order.go    # 客户端排序
│   │   ├── skill.go           # 技能模型
│   │   └── wecom_credentials.go
│   ├── handlers/              # HTTP 处理器
│   │   ├── wecom.go           # iLink 消息处理
│   │   ├── assistant.go       # AI 助手 API
│   │   ├── notification.go    # 通知管理
│   │   ├── alert.go           # 告警管理
│   │   ├── weather.go         # 天气配置
│   │   ├── skill.go           # 技能管理
│   │   ├── websocket.go       # WebSocket 处理
│   │   └── stats.go           # 统计接口
│   ├── services/              # 业务服务
│   │   ├── llm.go             # LLM 调用封装
│   │   ├── agent.go           # 意图判断
│   │   ├── chat_handler.go    # 聊天处理
│   │   ├── memory_handler.go  # 记忆处理
│   │   ├── reminder_handler.go # 提醒处理
│   │   ├── intent_recognizer.go # 意图识别
│   │   ├── time_parser.go     # 时间解析
│   │   └── cache.go           # 缓存服务
│   ├── agent/                 # Agent 适配器
│   │   ├── agent.go           # Agent 核心
│   │   ├── router.go          # 路由分发
│   │   ├── claude_adapter.go  # Claude 适配器
│   │   └── http_adapter.go    # HTTP 适配器
│   ├── skill/                 # 技能系统
│   │   ├── manager.go         # 技能管理
│   │   ├── loader.go          # 技能加载
│   │   ├── parser.go          # 技能解析
│   │   ├── tool_registry.go   # 工具注册
│   │   ├── prompt_builder.go  # Prompt 构建
│   │   ├── types.go           # 类型定义
│   │   └── router.go          # 技能路由
│   ├── notify/                # 通知服务
│   │   ├── service.go         # 通知服务核心
│   │   ├── alert.go           # 告警服务
│   │   ├── weather.go         # 天气推送
│   │   ├── reminder.go        # 提醒调度
│   │   ├── router.go          # 通知路由
│   │   ├── drivers/           # 通知渠道驱动
│   │   │   ├── driver.go
│   │   │   ├── ilink.go       # iLink (微信)
│   │   │   └── feishu.go      # 飞书
│   │   └── types/
│   │       └── message.go
│   ├── ilink/                 # iLink 协议实现
│   │   ├── client.go          # iLink 客户端
│   │   ├── auth.go            # 认证登录
│   │   ├── monitor.go         # 消息监听
│   │   └── types.go           # 类型定义
│   └── messaging/             # 消息处理
│       ├── sender.go
│       ├── handler.go
│       └── markdown.go
├── frontend/                   # 前端应用
│   ├── src/
│   │   ├── components/        # React 组件
│   │   │   ├── Dashboard.tsx  # 监控面板
│   │   │   ├── SystemSettings.tsx
│   │   │   ├── NotificationSettings.tsx
│   │   │   ├── WeatherSettings.tsx
│   │   │   ├── AlertSettings.tsx
│   │   │   ├── AssistantSettings.tsx
│   │   │   ├── SkillSettings.tsx
│   │   │   ├── settings/      # 设置子组件
│   │   │   └── dashboard/     # 面板子组件
│   │   ├── hooks/             # React Hooks
│   │   │   ├── useWebSocket.ts
│   │   │   └── useClientDrag.ts
│   │   ├── services/          # 前端服务
│   │   │   └── websocket.ts
│   │   ├── types/             # TypeScript 类型
│   │   └── lib/               # 工具函数
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── docker-compose.yml         # Docker 编排
├── CLAUDE.md                  # 项目说明（本文件）
└── README.md                  # 项目简介
```

---

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL
- **实时通信**: Gorilla WebSocket

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **样式**: Tailwind CSS
- **图表**: Recharts
- **UI 组件**: Radix UI + shadcn/ui

### 基础设施
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx（生产环境）

---

## 环境配置

### 必需环境变量

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=fluxpanel

# 服务配置
SERVER_PORT=8080
```

### AI 助手配置

```bash
# LLM 配置（通过管理界面配置，存储在数据库）
LLM_PROVIDER=openai
LLM_API_KEY=your-api-key
LLM_MODEL=gpt-4o-mini
LLM_BASE_URL=https://api.openai.com/v1
```

---

## 开发指南

### 本地开发启动

```bash
# 启动后端
cd backend
go mod tidy
go run main.go

# 启动前端
cd frontend
npm install
npm run dev
```

### 添加新技能

1. 在 `backend/skill/skills/` 目录创建技能 YAML 文件
2. 定义技能名称、描述、工具和 Prompt
3. 通过管理界面上传或安装技能

### 添加新通知渠道

1. 在 `backend/notify/drivers/` 实现驱动接口
2. 注册到通知服务
3. 在管理界面配置渠道

---

## API 概览

### 监控相关
- `POST /api/report` - 客户端数据上报
- `GET /api/summary` - 获取汇总统计
- `GET /api/events` - 获取事件列表
- `GET /ws` - WebSocket 连接

### 通知管理
- `GET/POST/PUT/DELETE /api/notifications/channels` - 通知渠道管理
- `GET/POST/PUT/DELETE /api/notifications/rules` - 通知规则管理

### AI 助手
- `GET/PUT /api/assistant/llm` - LLM 配置管理
- `GET/POST/PUT/DELETE /api/assistant/todos` - Todo 管理
- `GET/DELETE /api/assistant/memories` - 记忆管理
- `GET /api/assistant/reminders` - 提醒列表

### iLink 登录
- `GET /api/wecom/login/qrcode` - 获取登录二维码
- `GET /api/wecom/login/status` - 获取登录状态
- `DELETE /api/wecom/session` - 登出

### 技能管理
- `GET/POST/DELETE /api/skills` - 技能管理
- `PUT /api/skills/:id/enable` - 启用/禁用技能

---

## 部署说明

### Docker Compose 部署（推荐）

```bash
docker-compose up -d
```

### 生产环境注意事项

1. 配置 HTTPS 反向代理
2. 设置安全的数据库密码
3. 定期备份数据库
