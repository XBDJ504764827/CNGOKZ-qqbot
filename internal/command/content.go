package command

import (
	"github.com/tencent-connect/botgo/dto/message"
)

// normalizeCommandContent removes the QQ mention prefix that may be included in
// GROUP_AT_MESSAGE_CREATE and trims the command text.
func normalizeCommandContent(content string) string {
	return message.ETLInput(content)
}
