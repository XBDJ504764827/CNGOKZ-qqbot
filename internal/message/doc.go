// Package message 提供 QQ 消息的接收与发送能力。
//
// receiver 负责处理收到的用户消息（日志输出 → 未来的指令系统）；
// sender 负责发送频道 / 群 / 私聊消息，未来供 LumiAdmin 通过 HTTP 调用。
package message
