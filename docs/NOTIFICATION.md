# LumiBot 通知系统说明

本文档说明通知系统的工作流程、支持的事件、消息模板格式与后续扩展方案。

## 1. 通知流程

```
外部系统
  │  POST /api/v1/events（X-API-Key）
  ▼
Event API（internal/api/events.go：校验 + 补全）
  │
  ▼
Event Bus（internal/event：多订阅者分发）
  │
  ▼
Notification Service（internal/notification/service.go）
  │
  ├─ 1. 开关检查（NOTIFICATION_ENABLE）
  ├─ 2. 规则判断（internal/rule：事件类型 + 最低级别）
  ├─ 3. 冷却防刷（cooldown.go：窗口内重复事件只发一次）
  ├─ 4. 模板渲染（template.go：按事件类型独立模板）
  └─ 5. QQ 发送（message.Sender：私聊 / 频道）+ 通知日志
```

## 2. 支持事件与规则

| 事件类型 | 通知标题 | 最低级别 | 默认冷却 | 说明 |
| --- | --- | --- | --- | --- |
| `SERVER_OFFLINE` | 服务器异常 | info | `NOTICE_COOLDOWN` | 游戏服务器离线 |
| `SERVER_ONLINE` | 服务器恢复 | info | `NOTICE_COOLDOWN` | 游戏服务器恢复 |
| `SYSTEM_WARNING` | 系统警告 | warning | `NOTICE_COOLDOWN` | 系统资源 / 服务告警 |
| `FORUM_REPORT_CREATED` | 论坛举报 | warning | `NOTICE_COOLDOWN` | 论坛新举报 |
| `WHITELIST_REQUEST_CREATED` | 新白名单申请 | warning | `NOTICE_COOLDOWN` | LumiAdmin 白名单新申请（data：nickname / steamid64 / contact） |
| `ADMIN_ACTION` | 管理操作 | error | 无 | 审计用途，**默认不通知**（规则 `Enabled=false`） |

规则要点：
- 未配置规则的事件类型（如 `MESSAGE_RECEIVED`）**不通知**，仅记录事件日志
- 事件级别低于规则 `MinLevel` 时不通知（如 `SYSTEM_WARNING` 的 info 级事件）
- 规则为数据驱动（`internal/rule`），未来可通过配置/后台动态调整

## 3. 消息模板格式

模板位于 `internal/notification/template.go`，每个事件类型独立模板，禁止硬编码在业务代码中。

### 模板变量

| 变量 | 说明 |
| --- | --- |
| `{{.Title}}` | 事件标题 |
| `{{.Message}}` | 事件描述 |
| `{{.TimeText}}` | 事件时间（`2006-01-02 15:04`） |
| `{{.LevelText}}` | 级别中文（普通 / 警告 / 错误 / 严重） |
| `{{get .Data "key"}}` | 事件 data 字段取值，缺失时输出 `-` |

### 模板示例（SERVER_OFFLINE）

输入事件：

```json
{
  "source": "GameMonitor",
  "event_type": "SERVER_OFFLINE",
  "level": "critical",
  "title": "服务器离线",
  "message": "KZ服务器01停止响应",
  "data": {"server": "KZ-01", "status": "离线"}
}
```

输出 QQ 消息：

```
[服务器异常]

服务器: KZ-01
状态: 离线
时间: 2026-08-03 16:00

KZ服务器01停止响应
请管理员处理。
```

### 渲染兜底策略

1. 事件类型专属模板（渲染失败或未注册时降级）
2. 通用兜底模板（`[标题] + 级别 + 时间 + 消息`）
3. 纯文本拼接（极端情况，保证通知不丢失）

## 4. 通知渠道

| 渠道 | 状态 | 目标配置 | 说明 |
| --- | --- | --- | --- |
| `QQ_PRIVATE` | ✅ 已实现 | `NOTIFY_PRIVATE_TARGET`（管理员 QQ openid） | 默认渠道，告警优先 |
| `QQ_CHANNEL` | ✅ 已实现 | `NOTIFY_CHANNEL_TARGET`（子频道 ID） | 频道公告 |
| `EMAIL` | ⏳ 预留 | - | 邮件通知 |
| `WEBHOOK` | ⏳ 预留 | - | Webhook 通知 |

## 5. 通知日志

每次发送记录结构化日志（`internal/notification/service.go`）：

```
INFO  notification sent  {"notification_id": "...", "event_id": "...", "event_type": "SERVER_OFFLINE", "send_status": "sent", "channel": "QQ_PRIVATE"}
ERROR notification send failed  {"event_id": "...", "event_type": "...", "send_status": "failed", "error": "..."}
```

状态取值：`sent`（已发送）/ `failed`（发送失败）/ `skipped`（目标未配置等）。

## 6. 后续扩展方案

| 扩展点 | 现状 | 方案 |
| --- | --- | --- |
| 冷却存储 | 内存 Map（`MemoryCooldown`） | 实现 `Cooldown` 接口的 Redis 版本，多实例共享冷却状态 |
| 渠道扩展 | QQ_PRIVATE / QQ_CHANNEL | Notification.Channel 已建模，EMAIL / WEBHOOK 按渠道扩展 `send()` |
| 规则动态化 | 内置规则表 | 规则表改为配置/数据库驱动 |
| 通知持久化 | 无（仅日志） | 通知记录落库，供 LumiAdmin 后台查询 |
| 模板外置 | Go 代码内置 | 模板文件（.tmpl）或配置中心加载，免发版调整 |
| 目标管理 | 环境变量配置 | 管理员白名单接口（后续权限系统阶段） |
