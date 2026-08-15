package notification

import "github.com/XBDJ504764827/LumiBot/internal/event"

// LevelText 将事件级别映射为中文展示文本（用于通知模板）。
func LevelText(level string) string {
	switch level {
	case event.LevelInfo:
		return "普通"
	case event.LevelWarning:
		return "警告"
	case event.LevelError:
		return "错误"
	case event.LevelCritical:
		return "严重"
	default:
		return "普通"
	}
}
