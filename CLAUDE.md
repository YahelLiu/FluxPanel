# WeCom AI Assistant 模块

## 概述

企业微信 AI 助手模块，集成到 FluxPanel 中，共享现有的通知系统、用户体系和定时任务基础设施。

---

## 核心功能

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
- 到时间主动发送企业微信消息
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
用户在企业微信发送消息
        ↓
    系统收到消息
        ↓
   意图判断（Agent）
        ↓
   执行对应能力
        ↓
  结果回复到企业微信
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
| wecom_user_id | string | 企业微信用户ID |
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
回复到企业微信
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
主动发送企业微信消息
    ↓
标记 sent = true
```

---

## API 设计

### 企业微信回调接口
```
POST /api/wecom/callback
```
接收企业微信推送的消息

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

1. **企业微信通知**
   - 复用 `notify/wechat.go` 中的 `WechatWorkNotifier`
   - 复用 `SendAppMessage(userID, title, content, event)` 发送消息
   - 复用 `GetAccessToken()` 获取访问令牌

2. **通知渠道配置**
   - 复用 `NotificationChannel` 模型
   - 复用 `WechatWorkConfig` 配置结构

3. **定时任务**
   - 复用 `notify/weather.go` 的调度模式
   - 新增 `notify/reminder.go` 处理提醒调度

4. **数据库**
   - 复用现有 GORM 和数据库连接
   - 在 `database/db.go` 中添加新模型迁移

### 文件结构

```
backend/
├── handlers/
│   ├── wecom.go          # 企业微信回调处理（新增）
│   └── assistant.go      # AI 助手 API（新增）
├── models/
│   └── assistant.go      # AI 助手数据模型（新增）
├── services/
│   ├── agent.go          # 意图判断（新增）
│   └── llm.go            # LLM 调用（新增）
├── notify/
│   └── reminder.go       # 提醒调度（新增）
└── main.go               # 添加新路由
```

---

## 配置

在环境变量或配置文件中添加：

```
# LLM 配置
LLM_PROVIDER=openai
LLM_API_KEY=your-api-key
LLM_MODEL=gpt-4o-mini
LLM_BASE_URL=https://api.openai.com/v1  # 可选，用于其他兼容 API

# 企业微信配置
WECOM_CORP_ID=your-corp-id
WECOM_AGENT_ID=your-agent-id
WECOM_SECRET=your-secret
WECOM_TOKEN=your-token          # 回调验证用
WECOM_ENCODING_AES_KEY=your-key # 消息加解密用
```

---

## MVP 边界

### 第一版包含
- ✅ 单用户或少量用户
- ✅ 文字消息
- ✅ 简单记忆
- ✅ 简单 Todo
- ✅ 一次性提醒
- ✅ 最近对话上下文
- ✅ 企业微信收发

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

### P1 - 企业微信集成
1. 企业微信回调处理（`handlers/wecom.go`）
2. 企业微信消息发送
3. 消息加解密处理

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
