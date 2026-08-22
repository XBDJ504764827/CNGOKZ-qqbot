# LumiBot 配置说明

配置文件为项目根目录 `.env`（复制自 `configs/.env.example`），
优先级：进程环境变量 > `.env` 文件 > 内置默认值。

> 使用 `cp configs/.env.example .env` 生成你的配置文件。

## 配置项总览

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `ENV` | 是 | `dev` | 运行环境：`dev`（QQ 沙箱）/ `prod`（QQ 正式） |
| `QQ_APP_ID` | 是 | 空 | QQ 开放平台机器人 AppID |
| `QQ_SECRET` | 是 | 空 | QQ 开放平台机器人 AppSecret |
| `QQ_TOKEN` | 否 | 空 | 旧式 BotToken（新式鉴权用 Secret） |
| `BOT_DEBUG` | 否 | `false` | 开启 botgo SDK 调试日志（生产保持 false） |
| `HTTP_ADDR` | 否 | `:8080` | 内置 HTTP 服务监听地址 |
| `EVENT_API_KEYS` | 否 | 空（拒绝所有上报） | 事件上报 API Key，逗号分隔 |
| `EVENT_RATE_LIMIT` | 否 | `100` | 单 Key 每分钟事件上报上限 |
| `NOTIFICATION_ENABLE` | 否 | `true` | 是否启用 QQ 通知 |
| `NOTICE_COOLDOWN` | 否 | `300` | 同类型事件通知冷却（秒） |
| `NOTIFY_PRIVATE_TARGET` | 否 | 空（跳过私聊通知） | QQ_PRIVATE 渠道目标：管理员 QQ openid |
| `NOTIFY_CHANNEL_TARGET` | 否 | 空 | QQ_CHANNEL 渠道目标：子频道 ID |
| `LUMIADMIN_CALLBACK_URL` | 否 | 空（禁用按钮审批） | LumiAdmin 地址，用于 QQ 审批和 `/wl` 白名单状态查询 |
| `LUMIADMIN_QQ_TOKEN` | 否 | 空 | LumiAdmin QQ 集成令牌（与后端 `QQ_INTEGRATION_TOKEN` 一致），用于 QQ 审批和 `/wl` 查询鉴权 |
| `QQ_APPROVAL_AUDIT_PATH` | 否 | `logs/qq-approval-audit.jsonl` | 白名单 QQ 审批审计 JSONL 文件；生产环境配置到持久化卷 |
| `QQ_COMMAND_AUDIT_PATH` | 否 | `logs/qq-command-audit.jsonl` | QQ 指令审计 JSONL 文件（如 `/bind`）；生产环境配置到持久化卷 |
| `LOG_LEVEL` | 否 | `info` | 日志级别：debug / info / warn / error |

> 优先级：进程环境变量 > `.env` 文件 > 默认值。`.env` 文件仅作为默认值来源，
> 同名环境变量始终覆盖文件中的值。