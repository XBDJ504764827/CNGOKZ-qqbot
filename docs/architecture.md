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
 ├── handler.NewDefaultHandler(logger)   → handler.Handler
 ├── bot.NewClient(cfg, logger, handler) → bot.Client
 │     └── RegisterEvents(handler, logger) → dto.Intent
 └── api.NewServer(cfg.HTTP, logger)     → api.Server
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
                                        │ 转换为业务上下文
                                        ▼
                                 handler.Handler（接口）
                                        │
                                        ▼
                           DefaultHandler（第一阶段：日志）
                           │ 第二阶段：指令路由 / 回复消息（注入 OpenAPI）
```

## 5. HTTP 服务（LumiAdmin 预留）

| 路由 | 方法 | 当前状态 | 用途 |
| --- | --- | --- | --- |
| `/health` | GET | ✅ 已实现 | 健康检查 |
| `/api/v1/message/send` | POST | ⏳ 预留 | LumiAdmin 推送管理员通知 |

预留接口规划：请求鉴权（签名 / token）、消息体校验（管理员 ID、内容）、
复用 bot 层 OpenAPI 客户端发送群 / 私聊消息、失败重试与审计日志。

## 6. 日志规范

- 业务日志：zap 结构化输出，`ts/level/caller/msg/字段` 格式
- SDK 日志：通过 `BotgoAdapter` 桥接至同一 zap logger
- 生产环境（`ENV=prod`）输出 JSON，便于日志平台采集

## 7. 生命周期

```
启动：配置 → 日志 → 组件装配 → errgroup 并行启动 HTTP + QQ 网关
退出：SIGINT / SIGTERM → HTTP 优雅关闭（5s 超时）→ bot 连接停止 → 进程退出
```
