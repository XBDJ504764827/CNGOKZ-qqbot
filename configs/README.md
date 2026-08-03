# configs 目录说明

本目录存放 LumiBot 的配置模板与部署相关配置文件。

| 文件 | 用途 |
| --- | --- |
| `.env.example` | 环境变量配置模板。复制为项目根目录 `.env` 后填写真实凭证 |

## 配置项一览

| 环境变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `ENV` | 否 | `dev` | 运行环境：`dev`（QQ 沙箱）/ `prod`（QQ 正式） |
| `QQ_APP_ID` | 是 | - | QQ 开放平台机器人 AppID |
| `QQ_TOKEN` | 否 | - | 机器人 Token（预留，旧式 BotToken 鉴权） |
| `QQ_SECRET` | 是 | - | 机器人 AppSecret（新式 oauth2 鉴权） |
| `QQ_OPENAPI_TIMEOUT` | 否 | `5s` | openapi 请求超时 |
| `HTTP_ADDR` | 否 | `:8080` | 内置 HTTP 服务监听地址 |
| `HTTP_READ_TIMEOUT` | 否 | `5s` | HTTP 读超时 |
| `HTTP_WRITE_TIMEOUT` | 否 | `10s` | HTTP 写超时 |
| `HTTP_IDLE_TIMEOUT` | 否 | `60s` | HTTP 空闲超时 |
| `LOG_LEVEL` | 否 | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `BOT_DEBUG` | 否 | `false` | 机器人调试模式，开启 SDK 调试输出（生产环境保持 `false`） |
| `EVENT_API_KEYS` | 否 | 空（拒绝所有） | 事件上报接口 API Key，逗号分隔多个；未配置时 fail-closed |
| `NOTIFICATION_ENABLE` | 否 | `true` | 是否启用通知（事件 → QQ 通知） |
| `NOTICE_COOLDOWN` | 否 | `300` | 通知冷却时间（秒），同一事件类型窗口内重复事件只通知一次 |
| `NOTIFY_PRIVATE_TARGET` | 否 | 空（跳过私聊通知） | QQ_PRIVATE 渠道目标：管理员 QQ openid |
| `NOTIFY_CHANNEL_TARGET` | 否 | 空 | QQ_CHANNEL 渠道目标：子频道 ID |

> 优先级：进程环境变量 > `.env` 文件 > 默认值。`.env` 文件仅作为默认值来源，
> 同名环境变量始终覆盖文件中的值。
