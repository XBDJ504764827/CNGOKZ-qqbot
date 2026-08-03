// Package event 实现 CNGOKZ 统一事件系统。
//
// 架构定位：LumiBot 不仅是 QQ 机器人，更是社区事件通知中心。
// 外部系统（LumiForum / LumiAdmin / 游戏服务器监控 / QQ 自身）通过
// HTTP API（POST /api/v1/events）上报事件，经 Event Bus 分发到各订阅者
// （QQ 通知、日志记录、未来 Webhook 等）。
//
//	外部系统 → HTTP API → Event Bus → Subscriber → Notification Handler
//
// Bus 以接口定义（event.go），当前提供内存实现（bus.go），
// 未来可扩展 Redis Pub/Sub 实现（RedisBus），实现同一接口即可无缝替换，
// 业务层无需感知。
package event
