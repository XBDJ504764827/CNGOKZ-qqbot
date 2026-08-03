# LumiBot

LumiBot 是 CNGOKZ 社区生态中的 QQ 官方机器人服务，基于 Go 语言与腾讯官方
[botgo SDK](https://github.com/tencent-connect/botgo) 开发。

## 项目介绍

- **定位**：CNGOKZ 社区在 QQ 平台的服务入口，同时也是**社区事件通知中心**
- **架构**：botgo 长连接事件驱动 + 内置 HTTP 服务（事件上报 + LumiAdmin 回调预留）
- **事件系统**：外部系统（LumiForum / LumiAdmin / 游戏服务器监控）通过 `POST /api/v1/events` 上报事件，经 Event Bus 分发到 QQ 通知等订阅者
- **通知系统**：事件经规则判断（`internal/rule`）→ 模板渲染 → 冷却防刷 → QQ 私聊/频道通知（详见 [docs/NOTIFICATION.md](docs/NOTIFICATION.md)）
- **当前阶段**：通知系统（事件 → 管理员 QQ 通知）

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
├── .github/
│   └── workflows/
│       └── ci.yml             # CI 工作流（gofmt / vet / golangci-lint / test）
├── .golangci.yml               # golangci-lint 配置
├── cmd/
│   └── bot/
│       └── main.go            # 入口：依赖装配与生命周期管理
├── internal/
│   ├── config/                # 配置加载（.env + 环境变量）
│   ├── logger/                # zap 结构化日志 + botgo 日志适配
│   ├── bot/
│   │   ├── client.go          # Bot Client：openapi 工厂 + 生命周期
│   │   ├── gateway.go         # Gateway：websocket 长连接管理
│   │   ├── event.go           # QQ 事件注册（READY / MESSAGE_CREATE / AT_MESSAGE_CREATE）
│   │   └── handler.go         # QQ 事件业务分发（→ message 层）
│   ├── message/
│   │   ├── sender.go          # QQ 消息发送（频道 / 群 / 私聊）
│   │   └── receiver.go        # QQ 消息接收处理
│   ├── event/                 # 统一事件系统（社区事件通知中心）
│   │   ├── event.go           # Bus / Subscription 接口
│   │   ├── types.go           # Event 模型 + 事件类型 / 级别常量
│   │   ├── bus.go             # 内存事件总线（预留 Redis Pub/Sub 扩展）
│   │   └── handler.go         # Handler 接口 + 函数式适配器
│   ├── rule/                  # 通知规则系统（事件类型 + 级别 → 是否通知）
│   ├── notification/          # 通知系统：模型 / 模板 / 冷却 / 服务（事件 → QQ 通知）
│   └── api/                   # HTTP 服务：/health + /api/v1/events（事件上报）
├── configs/                   # 配置模板
├── docs/                      # 架构 / CI / API 文档
└── README.md
```

## 环境要求

| 依赖 | 版本 |
| --- | --- |
| Go | 1.24+ |
| QQ 开放平台机器人 | 已创建并获取 AppID / Secret |
| systemd（生产） | 任意主流发行版 |

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

## 统一事件系统（事件通知中心）

LumiBot 接收来自外部系统的事件并分发处理：

```
外部系统（LumiForum / LumiAdmin / 游戏服务器监控）
    │
    │  POST /api/v1/events（X-API-Key 鉴权）
    ▼
Event API Layer（internal/api/events.go：校验 + 补全字段）
    │
    ▼
Event Bus（internal/event：内存实现，预留 Redis Pub/Sub）
    │
    ▼
Subscriber（支持同一事件多个订阅者）
    │
    ▼
Notification Handler（internal/notification：当前记录日志，未来发 QQ 通知）
```

- **事件模型**：`source` / `event_type` / `level` / `title` / `message` / `data`，完整说明见 [docs/API.md](docs/API.md)
- **关键事件**：`SYSTEM_WARNING`、`SERVER_OFFLINE`、`FORUM_REPORT_CREATED` 已被通知处理器订阅
- **安全**：`X-API-Key` 校验（`EVENT_API_KEYS` 配置，未配置时拒绝所有上报）

## 通知系统

事件自动转换为管理员可读的 QQ 通知：

```
Event → 规则判断（rule）→ 模板渲染（template）→ 冷却防刷（cooldown）→ QQ 发送（message.Sender）
```

- **支持事件**：`SERVER_OFFLINE` / `SERVER_ONLINE` / `SYSTEM_WARNING` / `FORUM_REPORT_CREATED`（`ADMIN_ACTION` 默认关闭）
- **通知渠道**：`QQ_PRIVATE`（管理员私聊，需配置 `NOTIFY_PRIVATE_TARGET`）/ `QQ_CHANNEL`（频道），EMAIL / WEBHOOK 预留
- **防刷**：`NOTICE_COOLDOWN`（秒）内同一事件类型只通知一次，内存实现，预留 Redis
- **模板**：每事件独立模板（`internal/notification/template.go`），缺失字段安全兜底

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

## 生产部署（二进制 + systemd）

本项目采用传统二进制部署，不使用 Docker / Kubernetes：

```bash
# 1. 构建
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o lumibot ./cmd/bot

# 2. 上传到服务器 /opt/lumibot/，并放置 .env
scp lumibot user@server:/opt/lumibot/

# 3. systemd 管理（服务模板见 docs/CI.md）
systemctl enable --now lumibot
```

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
| 第六阶段 | CD 自动化：CI 产物 → 服务器二进制分发（二进制 + systemd 部署） |

CI 流程、分支规范与 Branch Protection 配置见 [docs/CI.md](docs/CI.md)，架构设计见 [docs/architecture.md](docs/architecture.md)。

## License

MIT
