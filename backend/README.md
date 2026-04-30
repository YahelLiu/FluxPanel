# FluxPanel Backend

企业微信 AI 助手后端服务。

## 致谢 / Acknowledgements

本项目参考并使用了以下开源项目的代码：

### [WeClaw](https://github.com/fastclaw-ai/weclaw)

> WeClaw - WeChat AI Agent Bridge — connect WeChat to AI agents (Claude, Codex, Gemini, Kimi, etc.)

本项目参考 WeClaw 项目实现了企业微信 iLink Bot API 的集成，包括：

- `ilink/` - iLink API 客户端实现（认证、消息收发、长轮询）
- `messaging/` - 消息发送与 Markdown 转换
- `agent/` - Agent 接口定义

**项目地址：** https://github.com/fastclaw-ai/weclaw

**License：** [MIT](https://github.com/fastclaw-ai/weclaw/blob/main/LICENSE)

---

## 功能特性

- **iLink Bot API 集成** - 扫码登录即可使用，无需企业微信管理员配置
- **AI 助手** - 支持多轮对话、记忆管理、Todo、提醒功能
- **消息收发** - 支持文本消息、typing 状态
- **定时提醒** - 通过 iLink API 主动发送提醒消息

## 快速开始

### 环境要求

- Go 1.26+
- PostgreSQL 15+

### 本地运行

```bash
# 安装依赖
go mod tidy

# 运行
go run .
```

### Docker 运行

```bash
cd ..
docker-compose up --build
```

## API 接口

### 微信登录

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/wecom/login/qrcode` | 获取登录二维码 |
| GET | `/api/wecom/login/status?qrcode=xxx` | 轮询登录状态 |
| GET | `/api/wecom/status` | 获取连接状态 |
| DELETE | `/api/wecom/session` | 登出 |

### AI 助手

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/wecom/chat` | 发送消息给 AI |
| POST | `/api/wecom/test` | 测试发送微信消息 |

## 配置

环境变量：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `DB_HOST` | 数据库地址 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户 | `postgres` |
| `DB_PASSWORD` | 数据库密码 | `postgres` |
| `DB_NAME` | 数据库名称 | `client_monitor` |

## License

MIT License
