# LumiBot

LumiBot 是 CNGOKZ 社区生态中的 QQ 官方机器人服务，基于 Go 语言与腾讯官方
[botgo SDK](https://github.com/tencent-connect/botgo) 开发。

## 项目介绍

- **定位**：CNGOKZ 社区在 QQ 平台的服务入口，负责事件通知、管理员提醒与机器人指令
- **架构**：botgo 长连接事件驱动 + 内置 HTTP 服务（供 LumiAdmin 后台回调）
- **当前阶段**：第一阶段 —— 基础架构初始化（配置 / 日志 / 事件注册 / HTTP 预留 / Docker）

```
┌──────────────┐   POST /api/message/send（预留）   ┌──────────────┐
│  LumiAdmin   │ ────────────────────────────────────▶ │              │
│  （后台系统） │                                       │   LumiBot    │
└──────────────┘                                       │              │
┌──────────────┐    QQ 网关 websocket 长连接（事件）    │  cmd/bot     │
│  QQ 开放平台 │ ────────────────────────────────────▶ │  + internal/ │
└──────────────┘                                       └──────────────┘
```

## 目录结构

```
.
├── cmd/
│   └── bot/
│       └── main.go            # 入口：依赖装配与生命周期管理
├── internal/
│   ├── config/                # 配置加载（.env + 环境变量）
│   ├── logger/                # zap 结构化日志 + botgo 日志适配
│   ├── bot/
│   │   ├── client.go          # Bot Client：openapi 工厂 + 生命周期
│   │   ├── gateway.go         # Gateway：websocket 长连接管理
│   │   ├── event.go           # 事件注册（READY / MESSAGE_CREATE / AT_MESSAGE_CREATE）
│   │   └── handler.go         # 业务分发（事件 → message 层）
│   ├── message/
│   │   ├── sender.go          # 消息发送（频道 / 群 / 私聊）
│   │   └── receiver.go        # 消息接收处理
│   └── api/                   # 内置 HTTP 服务（/health，LumiAdmin 预留）
├── configs/                   # 配置模板
├── docker/                    # Docker 部署说明
├── docs/                      # 架构文档
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 环境要求

| 依赖 | 版本 |
| --- | --- |
| Go | 1.24+ |
| Docker / docker compose | 任意近期版本（可选，容器化部署） |
| QQ 开放平台机器人 | 已创建并获取 AppID / Secret |

## QQ 机器人启动流程

```
main.go
  ↓
加载配置（.env + 环境变量）
  ↓
初始化 Logger（zap，注入 botgo SDK）
  ↓
创建 Bot Client（NewOpenAPI：沙箱/正式 + BOT_DEBUG）
  ↓
注册事件 Handler（READY / MESSAGE_CREATE / AT_MESSAGE_CREATE）
  ↓
连接 QQ Gateway（websocket 长连接，断线自动重连）
```

监听 `SIGINT` / `SIGTERM` 优雅关闭：关闭 HTTP 服务 → 断开 QQ 网关连接 → 释放资源。

## 已支持事件

| 事件 | 说明 | 处理位置 |
| --- | --- | --- |
| `READY` | 网关连接就绪（打印 bot 信息） | `bot/handler.go OnReady` |
| `MESSAGE_CREATE` | 频道消息（打印 user_id / channel_id / content） | `message/receiver.go` |
| `AT_MESSAGE_CREATE` | 频道内 @机器人 消息 | `message/receiver.go` |
| ERROR_NOTIFY | 网关连接异常（内部回调，记录错误日志） | `bot/handler.go OnError` |
| PLAIN | 未注册事件兑底（透传 debug 日志） | `bot/handler.go OnPlain` |

收到消息时日志示例：

```
INFO  message received  {"user_id": "xxxx", "guild_id": "yyyy", "channel_id": "zzzz", "content": "hello"}
```

## 开发说明

- **新增事件**：在 `internal/bot/event.go` 注册回调 → 在 `internal/bot/handler.go` 增加分发方法 → 在 `message` 层实现业务逻辑
- **发送消息**：注入 `message.Sender`（`SendChannelMessage` / `SendGroupMessage` / `SendC2CMessage`），未来供 LumiAdmin 通过 `POST /api/message/send` 调用
- **错误处理**：统一返回 error 并记录日志，禁止 panic（除启动失败）；启动失败由 main 输出 FATAL 退出
- **测试**：`go test ./...`（配置 / 事件注册 / 消息收发均有单测）

## 本地运行

```bash
# 1. 准备配置
cp configs/.env.example .env
#    编辑 .env，填写 QQ_APP_ID / QQ_SECRET

# 2. 运行
go run ./cmd/bot

# 3. 验证健康检查
curl http://127.0.0.1:8080/health
# {"status":"ok"}
```

## Docker 运行

```bash
cp configs/.env.example .env   # 填写 QQ_APP_ID / QQ_SECRET
docker compose up -d           # 构建并启动
docker compose logs -f         # 查看日志
docker compose down            # 停止
```

容器启动后同样通过 `http://127.0.0.1:8080/health` 验证。

## QQ 机器人配置说明

1. 前往 [QQ 开放平台](https://q.qq.com) 创建机器人，获取 **AppID** 与 **Secret**
2. 按需在开放平台开启事件订阅：
   - 频道：`AT_MESSAGE_CREATE`（@机器人消息）
   - 群聊：`GROUP_AT_MESSAGE_CREATE`
   - 私聊：`C2C_MESSAGE_CREATE`
3. 填入 `.env`：
   - `QQ_APP_ID`、`QQ_SECRET` 必填
   - `QQ_TOKEN` 预留（旧式 BotToken 鉴权）
   - `ENV=dev` 连接沙箱，`ENV=prod` 连接正式环境
   - `BOT_DEBUG=true` 开启 SDK 调试输出（生产环境保持 `false`）

## 后续开发规划

| 阶段 | 内容 |
| --- | --- |
| 第三阶段 | 指令系统：解析消息 → 指令路由 → 回复（通过 `message.Receiver` 扩展） |
| 第四阶段 | 与 LumiAdmin 通信：`POST /api/message/send` 推送管理员通知，接口鉴权 |
| 第五阶段 | 事件通知：论坛 / 服务器 / 管理事件订阅与推送到管理员 QQ |
| 第六阶段 | 运维完善：指标采集、分布式 session 管理、CI/CD |

详细设计见 [docs/architecture.md](docs/architecture.md) 与 [docker/README.md](docker/README.md)。

## License

MIT
