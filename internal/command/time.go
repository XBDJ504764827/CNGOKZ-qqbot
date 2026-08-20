package command

import (
	"strings"
	"time"
)

// beijingLocation 固定使用北京时间，避免机器人服务器时区配置影响用户看到的时间。
var beijingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func formatBeijingTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.In(beijingLocation).Format("2006-01-02 15:04:05")
		}
	}
	return raw
}
