# `/ban` 封禁信息查询指令

## 用法

```text
/ban <steamid64/steamid2>
```

支持 SteamID64、SteamID2 和 Steam 个人主页 URL，解析方式与 `/wl` 一致；帮助文案只宣传 SteamID64 / SteamID2。

所有用户均可在 QQ 私聊或群聊中查询。查询数据来自 LumiAdmin 本地数据库，不会在指令请求时直接访问 KZTimer Global API。

## 查询结果

查询结果始终包含两个区块：

- 网站封禁：`ban_records` 中 `source != global_ban` 的全部历史记录；
- 全球封禁：LumiAdmin 本地同步表 `global_bans` 中的全部历史记录，包括已过期记录。

两类记录均按时间倒序展示，最多显示各自最近 10 条，超过时会提示未展示的历史记录数量。

网站封禁字段：

```text
状态：🔒 封禁中 / ⌛ 已过期 / ✅ 已解除
封禁类型：Steam 账号封禁 / IP 封禁
原因：...
封禁时间：北京时间
到期时间：北京时间 / 永久
解封时间：北京时间（已解除时）
解封原因：...（有记录时）
```

全球封禁字段：

```text
状态：🌍 生效中 / ⌛ 已过期
封禁类型：作弊等 KZTimer 类型
原因：优先使用 notes；notes 为空时使用 stats
封禁时间：北京时间
到期时间：北京时间 / 永久
```

如果管理员已在网站本地解除全球封禁对应的本地封禁，仍显示：

```text
全球封禁状态：🌍 全球封禁中
备注：本地服务器已解除该封禁
```

如果没有记录：

```text
网站封禁：✅ 未发现封禁
全球封禁：✅ 未发现封禁
```

## 内部 API

```http
GET /api/integration/qq/ban/status?steam_input=<url-encoded-steam-input>
X-QQ-Token: <QQ_INTEGRATION_TOKEN>
```

返回：

```json
{
  "steamid64": "76561198012345678",
  "steamid": "STEAM_0:1:12345",
  "local_bans": [],
  "global_bans": []
}
```

LumiBot 查询失败时，对用户统一回复：

```text
查询封禁信息失败，请稍后重试。
```
