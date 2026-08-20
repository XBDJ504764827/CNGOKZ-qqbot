# `/wl` 白名单状态查询指令

## 用法

```text
/wl <steamid64/steamid2>
```

支持：

- SteamID64，例如 `76561198012345678`
- SteamID2，例如 `STEAM_0:1:12345`
- Steam 个人主页 URL，例如 `https://steamcommunity.com/profiles/76561198012345678`

帮助文案只宣传 SteamID64 / SteamID2；个人主页 URL 由 LumiAdmin 复用现有 Steam 标识解析能力支持。

## 使用范围

- QQ 私聊机器人；
- QQ 群聊中发送 `/wl ...`。

注意：当前使用的 botgo v0.2.1 / QQ 官方开放平台仅提供 `GROUP_AT_MESSAGE_CREATE` 群消息事件，实际接收群消息通常仍需要 @机器人；代码侧不会把 @ 标记当作指令参数。若要实现完全不 @ 即接收所有群消息，需要 QQ 平台提供普通群消息事件或更换支持该能力的接入方式。所有用户均可查询，不需要管理员授权。

## 回复规则

单条记录示例：

```text
玩家：76561198012345678

白名单状态：✅ 已通过
```

状态映射：

| LumiAdmin 状态 | QQ 回复 |
| --- | --- |
| `approved` | `✅ 已通过` |
| `pending` | `⏳ 待审核` |
| `rejected` | `❌ 已拒绝`，追加拒绝原因 |
| `revoked` | `⚠️ 已撤销` |
| 无记录 | `⚪ 未找到记录，该玩家可能未申请白名单` |

拒绝原因为空时显示：

```text
原因：未填写拒绝原因
```

同一 SteamID 存在多条历史记录时全部返回，并为每条记录显示对应的状态时间。时间优先使用状态变更时间（通过、拒绝或撤销），否则使用申请时间。

LumiAdmin 查询失败时，对用户统一回复：

```text
查询白名单状态失败，请稍后重试。
```

具体错误只记录在 LumiBot 日志中。

## 内部接口

LumiBot 调用 LumiAdmin：

```http
GET /api/integration/qq/whitelist/status?steam_input=<url-encoded-steam-input>
X-QQ-Token: <QQ_INTEGRATION_TOKEN>
```

该接口返回全部历史记录，但不返回联系方式、审核人等后台敏感信息。