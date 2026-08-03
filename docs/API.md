# LumiBot HTTP API 文档

本文档面向外部系统（LumiForum / LumiAdmin / 游戏服务器监控等），
说明如何向 LumiBot 上报社区事件。

## 接口总览

| 接口 | 方法 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/events` | POST | X-API-Key | 事件上报（统一事件系统入口） |
| `/health` | GET | 无 | 健康检查 |

## 1. POST /api/v1/events — 事件上报

外部系统产生事件（举报、告警、服务器离线、管理操作等）时调用，
事件经 Event Bus 分发到各订阅者（QQ 通知、日志等）。

### 请求 Header

| Header | 必填 | 说明 |
| --- | --- | --- |
| `Content-Type` | 是 | `application/json` |
| `X-API-Key` | 是 | 服务端配置的 `EVENT_API_KEYS` 之一（逗号分隔可配置多个） |

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
| `source` | string | **是** | 事件来源系统，如 `LumiForum` / `LumiAdmin` / `GameServer` / `QQ` |
| `event_type` | string | **是** | 事件类型，取值见下表 |
| `level` | string | 否 | `info` / `warning` / `error` / `critical` |
| `timestamp` | string (RFC3339) | 否 | 事件发生时间 |
| `title` | string | 否 | 标题，用于通知展示 |
| `message` | string | 否 | 事件描述 |
| `data` | object | 否 | 业务附加数据（任意 JSON） |

### 事件类型列表（event_type）

| 事件类型 | 来源 | 级别建议 | 说明 |
| --- | --- | --- | --- |
| `SYSTEM_WARNING` | LumiAdmin | warning | 系统警告（磁盘、资源等） |
| `SERVER_OFFLINE` | GameServer | critical | 游戏服务器离线 |
| `FORUM_REPORT_CREATED` | LumiForum | warning | 论坛新举报 |
| `ADMIN_ACTION` | LumiAdmin | info | 管理操作记录 |

> 其他事件类型可自由上报，LumiBot 全部接收并记录日志；
> 上述类型已被通知处理器订阅（未来触发 QQ 通知）。

### 响应

成功（202 Accepted）：

```json
{
  "status": "accepted",
  "event_id": "5f9d4c2a-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

失败：

| 状态码 | 场景 | 响应体 |
| --- | --- | --- |
| `400` | JSON 解析失败 / 缺少必填字段 / level 非法 | `{"error":"..."}` |
| `401` | API Key 缺失或错误 | `{"error":"unauthorized"}` |
| `500` | 事件发布失败 | `{"error":"event publish failed"}` |

### 调用示例

```bash
curl -X POST http://127.0.0.1:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{
    "source": "GameServer",
    "event_type": "SERVER_OFFLINE",
    "level": "critical",
    "title": "游戏服务器离线",
    "message": "mc-01 心跳超时，请尽快处理",
    "data": {"server": "mc-01", "region": "cn"}
  }'
```

服务端日志输出示例：

```
INFO  event received  {"event_id": "...", "source": "GameServer", "event_type": "SERVER_OFFLINE", "level": "critical"}
INFO  收到事件，准备发送QQ通知（通知发送待后续阶段实现）  {"event_id": "...", "source": "GameServer", "event_type": "SERVER_OFFLINE", ...}
```

## 安全说明

- 当前阶段：`X-API-Key` 简单校验（配置于 `EVENT_API_KEYS`，逗号分隔多个 Key）
- **未配置 Key 时拒绝所有上报请求**（fail-closed，防止误暴露）
- 后续扩展：JWT、请求签名验证、IP 白名单
