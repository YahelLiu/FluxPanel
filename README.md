# 客户端数据上报监控系统

一个轻量级的客户端数据监控面板，支持实时数据上报和可视化展示。

## 技术栈

- **后端**: Go + Gin + GORM
- **前端**: React + Vite + Tailwind CSS + Recharts
- **数据库**: PostgreSQL
- **实时通信**: WebSocket
- **部署**: Docker Compose

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

### POST /api/report

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

### GET /api/summary

获取汇总统计数据

### GET /api/events

获取事件列表（支持分页和筛选）

参数：
- `page`: 页码
- `page_size`: 每页数量
- `client_id`: 客户端ID筛选
- `status`: 状态筛选
- `event_type`: 事件类型筛选

### GET /api/stats/hourly

获取今日每小时事件统计

### GET /api/stats/clients

获取客户端事件排名

### WebSocket /ws

实时推送新事件

## 数据结构

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

## 功能特性

- ✅ 实时数据上报
- ✅ WebSocket 实时推送
- ✅ 数据可视化图表
- ✅ 事件列表展示
- ✅ 状态统计
- ✅ Docker 一键部署

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
