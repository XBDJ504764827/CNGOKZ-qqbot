# `/bind` QQ OpenID 获取指令

## 用法

用户在 QQ 私聊机器人发送：

```text
/bind
```

机器人回复：

```text
你的 QQ OpenID：

1234567890xxxxxxxx

请复制此 OpenID 到网站的 QQ 绑定页面。
```

## 使用规则

- 只处理 QQ C2C 私聊消息；
- 群聊、频道中发送 `/bind` 不会回复；
- 只支持精确指令 `/bind`，不接受参数和别名；
- 指令大小写不敏感；
- 所有 QQ 用户均可使用；
- OpenID 直接取自 QQ 私聊事件的消息发送者 ID；
- 每个用户每分钟最多使用 5 次；
- 用户可以重复执行，OpenID 不会变化；
- OpenID 冲突由管理员在网站绑定时人工处理，本次不自动查询或修改 LumiAdmin 用户信息。

无法获取 OpenID 时统一回复：

```text
暂时无法获取你的 QQ OpenID，请稍后重试。
```

## 审计

`/bind` 使用记录独立写入 `QQ_COMMAND_AUDIT_PATH` 配置的 JSONL 文件，默认路径为：

```text
logs/qq-command-audit.jsonl
```

示例：

```json
{
  "timestamp": "2026-08-20T05:30:00Z",
  "event": "command_bind",
  "command": "/bind",
  "openid": "1234567890xxxxxxxx",
  "source": "c2c",
  "result": "success"
}
```

审计文件权限为仅文件所有者可读写（`0600`）。