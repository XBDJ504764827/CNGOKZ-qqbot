# LumiBot HTTP API 文档

本文档面向 CNGOKZ 生态外部系统（**LumiAdmin / LumiForum / GameMonitor**），
说明如何向 LumiBot（事件接收中心）上报事件。

## 接口总览

| 接口 | 方法 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/events` | POST | X-API-Key | 事件上报（统一事件系统入口） |
| `/api/integration/qq/whitelist/status` | GET | X-QQ-Token | LumiAdmin 提供的 QQ 查询白名单状态接口（全部历史记录） |
| `/health` | GET | 无 | 健康检查 |

## 1. GET /api/integration/qq/whitelist/status — QQ 查询白名单状态

供 LumiBot 的 `/wl` 指令调用。请求需要使用与 QQ 审批相同的 `X-QQ-Token` Header，也接受 `Authorization: Bearer <token>`。

请求示例：

```bash
curl -G http://127.0.0.1:3001/api/integration/qq/whitelist/status \\
  -H 'X-QQ-Token: integration-secret' \\
  --data-urlencode 'steam_input=STEAM_0:1:12345'
```

响应返回解析后的 SteamID64 和全部历史白名单记录：

```json
{
  "steamid64": "76561198012345678",
  "steamid": "STEAM_0:1:12345",
  "items": [
    {
      "id": "uuid",
      "steamid64": "76561198012345678",
      "steamid": "STEAM_0:1:12345",
      "status": "rejected",
      "applied_at": "2026-08-01T10:00:00Z",
      "rejected_at": "2026-08-01T11:00:00Z",
      "rejection_reason": "信息不完整"
    }
  ]
}
```

支持 SteamID64、SteamID2 和 Steam 个人主页 URL。无记录时 `items` 为空数组。接口不会返回联系方式、审核人等敏感字段。

| HTTP 状态 | 场景 |
| --- | --- |
| `200` | 查询成功，可能没有记录 |
| `400` | Steam 标识格式错误或参数无效 |
| `401` | 集成令牌缺失或无效 |
| `500` | 数据库查询失败 |

## 2. POST /api/v1/events — 事件上报

外部系统产生事件（举报、告警、服务器离线、管理操作等）时调用，
事件经 Event Bus 分发到各订阅者（QQ 通知、日志等）。

### 请求 Header

| Header | 必填 | 说明 |
| --- | --- | --- |
| `Content-Type` | 是 | `application/json` |
| `X-API-Key` | 是 | 服务端分配的 API Key（见「安全设计」） |

### 请求 Body（统一事件模型）

```json
{
  "id": "uuid（可选，省略时服务端生成）",
  "source": "LumiForum（必填，事件来源系统）",
  "event_type": "FORUM_REPORT_CREATED（必填）",
  "level": "warning（可选：info / warning / error / critical，默认 info）",
  "timestamp": "2026-08-03T12:00:00+08:00（可选，默认服务端接收时间）",
  "title": "新举报（可选，通知标题）",
  "message": "发现违规帖子（可选，事件描述）",
  "data": { "任意": "业务附加数据，可选" }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 否 | 事件唯一标识（UUID），省略时服务端生成 |
| `source` | string | **是** | 事件来源系统：`LumiAdmin` / `LumiForum` / `GameMonitor` / `QQ` |
| `event_type` | string | **是** | 事件类型，取值见下表 |
| `level` | string | 否 | `info` / `warning` / `error` / `critical` |
| `timestamp` | string (RFC3339) | 否 | 事件发生时间 |
| `title` | string | 否 | 标题，用于通知展示 |
| `message` | string | 否 | 事件描述 |
| `data` | object | 否 | 业务附加数据（任意 JSON） |

### 事件类型列表（event_type）

**通用事件：**

| 事件类型 | 适用来源 | 级别建议 | 说明 | 默认通知 |
| --- | --- | --- | --- | --- |
| `SYSTEM_WARNING` | LumiAdmin | warning | 系统警告（磁盘、资源等） | ✅ |
| `SERVER_OFFLINE` | GameMonitor | critical | CS / KZ 服务器离线 | ✅ |
| `SERVER_ONLINE` | GameMonitor | info | 服务器恢复 | ✅ |
| `FORUM_REPORT_CREATED` | LumiForum | warning | 论坛新举报 | ✅ |
| `ADMIN_ACTION` | LumiAdmin | info | 管理操作记录 | ❌（默认仅记录） |
| `WHITELIST_REQUEST_CREATED` | LumiAdmin | warning | 白名单新申请（等待审核） | ✅ |

**论坛通知事件（LumiForum 站内通知 → QQ，对应 LumiForum `NotificationEvent`）：**

| 事件类型 | 对应站内事件 | 级别建议 | 默认通知 | data 约定字段 |
| --- | --- | --- | --- | --- |
| `FORUM_COMMENT_REPLIED` | CommentReplied | info | ✅ | `target_openid` `topic_title` `actor_name` `topic_id` `topic_slug` `comment_id` |
| `FORUM_COMMENT_CREATED` | CommentCreated | info | ✅ | `target_openid` `topic_title` `actor_name` `topic_id` `topic_slug` `comment_id` |
| `FORUM_TOPIC_LIKED` | TopicLiked | info | ✅ | `target_openid` `topic_title` `actor_name` `topic_id` `topic_slug` |
| `FORUM_COMMENT_LIKED` | CommentLiked | info | ✅ | `target_openid` `topic_title` `actor_name` `topic_id` `comment_id` |
| `FORUM_TOPIC_FAVORITED` | TopicFavorited | info | ✅ | `target_openid` `topic_title` `actor_name` `topic_id` `topic_slug` |
| `FORUM_USER_FOLLOWED` | UserFollowed | info | ✅ | `target_openid` `actor_name` |
| `FORUM_POLL_VOTED` | PollVoted | info | ✅ | `target_openid` `topic_title` `poll_title` `actor_name` `topic_id` `topic_slug` |
| `FORUM_REPORT_PROCESSED` | ReportProcessed | warning | ✅ | `target_openid` `report_id` `result` |

> 其他事件类型可自由上报（必填字段校验通过即可），LumiBot 全部接收并记录日志；
> 是否触发 QQ 通知由通知规则决定（见 docs/NOTIFICATION.md）。

### 通知目标指定（data 约定）

通知发给谁由 `data.target_openid` 指定（网站管理员在后台配置用户 QQ 绑定后，由网站写入）：

| data 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `target_openid` | string | 否 | 接收通知的 QQ 用户 openid（单目标，论坛事件）；省略时发给默认管理员 |
| `openids` | string[] | 否 | 接收通知的 QQ 用户 openid 列表（多目标，如白名单申请）；**省略/空时发给默认管理员**（`NOTIFY_PRIVATE_TARGET`） |
| `actor_name` | string | 否 | 触发动作的用户昵称（模板展示用） |
| `topic_title` | string | 否 | 主题标题（模板展示用） |
| `topic_id` / `topic_slug` / `comment_id` / `poll_title` / `report_id` / `result` | string | 否 | 业务标识与结果（模板展示用） |
| 其他 | 任意 | 否 | 自定义字段，LumiBot 原样保留 |

### 统一响应格式

**成功（HTTP 202 Accepted）：**

```json
{
  "success": true,
  "event_id": "5f9d4c2a-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

**失败（HTTP 400 / 401 / 429 / 500）：**

```json
{
  "success": false,
  "error": "错误原因"
}
```

| HTTP 状态 | 场景 |
| --- | --- |
| `202` | 事件受理成功 |
| `400` | 参数错误（JSON 解析失败 / 缺少必填字段 / level 非法） |
| `401` | 认证失败（X-API-Key 缺失或错误，或服务端未配置 Key） |
| `429` | 触发限流（单来源超过 `EVENT_RATE_LIMIT` 次/分钟） |
| `500` | 服务器内部错误（事件发布失败） |

### 调用示例

```bash
curl -X POST http://127.0.0.1:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "X-API-Key: key-monitor" \
  -d '{
    "source": "GameMonitor",
    "event_type": "SERVER_OFFLINE",
    "level": "critical",
    "title": "游戏服务器离线",
    "message": "KZ-01 心跳超时，请尽快处理",
    "data": {"server": "KZ-01", "region": "cn"}
  }'
```

成功响应：

```json
{"success": true, "event_id": "5f9d4c2a-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}
```

服务端日志：

```
INFO  event received  {"event_id": "...", "source": "GameMonitor", "event_type": "SERVER_OFFLINE", "level": "critical"}
INFO  notification sent  {"notification_id": "...", "event_id": "...", "event_type": "SERVER_OFFLINE", "send_status": "sent", "channel": "QQ_PRIVATE"}
```

## 2. 外部系统接入流程

```
LumiAdmin / LumiForum / GameMonitor
        │  HTTP POST /api/v1/events（X-API-Key）
        ▼
    LumiBot（事件接收中心）
        │
        ▼
   Event Bus → Notification → QQ 管理员
```

1. **获取 API Key**：联系 LumiBot 管理员，在 `EVENT_API_KEYS` 中分配专属 Key
   （建议每个系统独立：`key-admin` / `key-forum` / `key-monitor`）
2. **确认事件协议**：按上表选择 `source` / `event_type` / `level` 及 `data` 字段
3. **上报事件**：调用 `POST /api/v1/events`，校验 `success=true` 即受理成功
4. **后续接入**：Go 项目可直接使用预留 SDK（`pkg/sdk`，见下文）

## 3. Go SDK（预留）

`pkg/sdk` 提供官方 Go SDK 接口设计（后续版本提供 HTTP 实现），
外部系统可提前按协议常量接入：

```go
import "github.com/XBDJ504764827/LumiBot/pkg/sdk"

// 协议常量：sdk.EventServerOffline / sdk.SourceGameMonitor / sdk.LevelCritical ...
req := &sdk.SendEventRequest{
    Source:    sdk.SourceGameMonitor,
    EventType: sdk.EventServerOffline,
    Level:     sdk.LevelCritical,
    Title:     "游戏服务器离线",
    Message:   "KZ-01 心跳超时",
    Data:      map[string]interface{}{"server": "KZ-01"},
}
// client := sdk.NewHTTPClient("http://127.0.0.1:8080", "key-monitor") // 下一阶段提供
// resp, err := client.SendEvent(ctx, req)
```

## 4. 安全设计

| 层次 | 机制 | 配置 |
| --- | --- | --- |
| 认证 | `X-API-Key` Header 校验（支持多 Key，分别分配给不同系统） | `EVENT_API_KEYS=key-admin,key-forum,key-monitor` |
| 限流 | 按 API Key 固定窗口限流（默认 100 次/分钟，`0` 关闭） | `EVENT_RATE_LIMIT=100` |
| 兜底 | 未配置任何 Key 时拒绝所有请求（fail-closed） | - |

- 当前阶段：简单 API Key 校验；后续扩展 JWT、HMAC 签名验证、IP 白名单
- 限流与冷却的区别：限流保护 API（防异常系统刷请求）；冷却去重通知（防重复骚扰管理员）

## 5. 错误排查

| 现象 | 可能原因 |
| --- | --- |
| 401 `invalid api key` | Key 错误 / 未配置 `EVENT_API_KEYS` |
| 429 `rate limit exceeded` | 单来源超过每分钟上限，等待窗口重置或调大 `EVENT_RATE_LIMIT` |
| 400 `source 不能为空` | 缺少必填字段，检查请求 Body |
| 202 但未收到 QQ 通知 | 事件类型未订阅 / 规则未命中 / 通知开关关闭 / 冷却期内（查看服务端日志） |
