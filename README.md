# LumiBot

LumiBot 是 CNGOKZ 社区生态中的 QQ 官方机器人服务，基于 Go 语言与腾讯官方
[botgo SDK](https://github.com/tencent-connect/botgo) 开发。

## 项目介绍

- **定位**：CNGOKZ 社区在 QQ 平台的服务入口，负责事件通知、管理员提醒与机器人指令
- **架构**：botgo 长连接事件驱动 + 内置 HTTP 服务（供 LumiAdmin 后台回调）
- **当前阶段**：第一阶段 —— 基础架构初始化（配置 / 日志 / 事件注册 / HTTP 预留 / Docker）

```
┌──────────────┐   POST /api/v1/message/send（预留）   ┌──────────────┐
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
│   ├── bot/                   # botgo 客户端：client.go / event.go
│   ├── handler/               # 消息处理接口与默认实现
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

## 后续开发规划

| 阶段 | 内容 |
| --- | --- |
| 第二阶段 | 指令系统：解析消息 → 指令路由 → 回复（通过实现 `handler.Handler` 扩展） |
| 第三阶段 | 与 LumiAdmin 通信：`POST /api/v1/message/send` 推送管理员通知，接口鉴权 |
| 第四阶段 | 事件通知：论坛 / 服务器 / 管理事件订阅与推送到管理员 QQ |
| 第五阶段 | 运维完善：指标采集、分布式 session 管理、CI/CD |

详细设计见 [docs/architecture.md](docs/architecture.md) 与 [docker/README.md](docker/README.md)。

## License

MIT
