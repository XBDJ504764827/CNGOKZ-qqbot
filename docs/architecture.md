# LumiBot 架构设计

## 1. 设计原则

- **依赖注入**：`cmd/bot/main.go` 统一装配组件，各包通过构造函数接收依赖（配置、日志、处理器），不隐式依赖全局状态
- **分层隔离**：bot 层只负责 botgo SDK 适配，handler 层只负责业务消息处理，api 层只负责 HTTP，互不感知
- **禁止硬编码**：所有可配置项（端口、超时、日志级别等）集中于 `internal/config`，默认值统一管理
- **扩展优先**：新增事件类型只需在 `internal/bot/event.go` 注册并路由到 `handler.Handler` 接口

## 2. 组件依赖关系

```
main (cmd/bot)
 ├── config.Load()          → *config.Config
 ├── logger.New()           → *zap.Logger
 │     └── NewBotgoAdapter  → botgo.SetLogger（SDK 日志统一）
 ├── bot.NewOpenAPI(cfg)    → openapi.OpenAPI（沙箱/正式 + BOT_DEBUG）
 ├── message.NewSender(api) → *message.Sender（频道/群/私聊发送）
 ├── message.NewReceiver()  → *message.Receiver（消息接收处理）
 │     └── OnCommand        → command.Router（/bind 指令分发）
 ├── bot.NewHandler(...)    → *bot.Handler（QQ 事件业务分发）
 ├── bot.NewClient(...)     → *bot.Client
 │     └── Start：WS 网关信息 → RegisterEvents → Gateway.Start（自动重连）
 ├── event.NewMemoryBus()   → *event.MemoryBus（统一事件总线）
 ├── notification.NewHandler() → 订阅 SYSTEM_WARNING / SERVER_OFFLINE / FORUM_REPORT_CREATED
 └── api.NewServer(cfg, bus)     → api.Server（/health + /api/v1/events）
```

## 3. 配置系统

| 来源 | 优先级 |
| --- | --- |
| 进程环境变量 | 高 |
| `.env` 文件（godotenv 加载，项目根目录） | 中 |
| 内置默认值（`config.go` 集中定义） | 低 |

## 4. 事件流

```
QQ 网关 ──websocket──▶ botgo SDK ──▶ internal/bot/event.go（注册回调）
                                        │ 适配为业务事件
                                        ▼
                                 internal/bot/handler.go（业务分发）
                                        │
                                        ▼
                             internal/message（接收处理 / 发送能力）
```

## 5. 消息发送

`internal/message/sender.go` 依赖最小接口 `MessageAPI`（仅声明三个发送方法），
隔离 botgo 完整接口，便于测试 mock；`openapi.OpenAPI` 天然满足该接口。

| 方法 | 场景 | 未来调用方 |
| --- | --- | --- |
| `SendChannelMessage` | QQ 频道消息 | LumiAdmin |
| `SendGroupMessage` | 群消息 | LumiAdmin 通知 |
| `SendC2CMessage` | 私聊 / 管理员通知 | LumiAdmin 通知 / `/bind` 指令回复 |

## 6. HTTP 服务

| 路由 | 方法 | 当前状态 | 用途 |
| --- | --- | --- | --- |
| `/health` | GET | ✅ 已实现 | 健康检查 |
| `/api/v1/events` | POST | ✅ 已实现 | 外部系统事件上报（X-API-Key 鉴权） |
| `/api/message/send` | POST | ⏳ 预留 | LumiAdmin 推送管理员通知 |

## 7. 统一事件系统

```
外部系统 → POST /api/v1/events（X-API-Key）
        → api/events.go（解析 JSON → Validate → Normalize）
        → event.Bus（接口；MemoryBus 实现，预留 Redis Pub/Sub）
        → 订阅者（event.Handler；同类型多订阅者）
        → notification.Handler（当前日志，未来 QQ 通知）
```

- **接口隔离**：`event.Bus` / `event.Handler` / `event.Subscription` 均为接口，业务层不依赖具体实现
- **扩展点**：新增订阅者实现 `event.Handler` 后 `bus.Subscribe(type, handler)` 即可；多实例部署时替换 `RedisBus` 实现，业务代码不变
- **安全**：`X-API-Key` 简单校验（`EVENT_API_KEYS`，fail-closed）；后续扩展 JWT / 签名验证

## 8. 日志规范

- 业务日志：zap 结构化输出，`ts/level/caller/msg/字段` 格式
- SDK 日志：通过 `BotgoAdapter` 桥接至同一 zap logger
- 生产环境（`ENV=prod`）输出 JSON，便于日志平台采集

## 9. 生命周期

```
启动：配置 → 日志 → 组件装配 → errgroup 并行启动 HTTP + QQ 网关
退出：SIGINT / SIGTERM → HTTP 优雅关闭（5s 超时）→ bot 连接停止 → 进程退出
```
