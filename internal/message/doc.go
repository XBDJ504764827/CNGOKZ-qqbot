// Package message 提供 QQ 消息的接收与发送能力。
//
// receiver 负责处理收到的用户消息（指令分发 → 普通文本日志）；
// sender 负责发送频道 / 群 / 私聊消息，未来供 LumiAdmin 通过 HTTP 调用。
package message
