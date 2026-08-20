package bot

import (
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// TestRegisterEvents_Intents 验证事件注册返回的 intent 包含预期事件。
func TestRegisterEvents_Intents(t *testing.T) {
	intents := RegisterEvents(newTestHandler(), zap.NewNop())

	if intents == 0 {
		t.Fatal("RegisterEvents() returned zero intents")
	}
	if intents&dto.EventToIntent(dto.EventMessageCreate) == 0 {
		t.Error("intents missing MESSAGE_CREATE")
	}
	if intents&dto.EventToIntent(dto.EventAtMessageCreate) == 0 {
		t.Error("intents missing AT_MESSAGE_CREATE")
	}
	if intents&dto.EventToIntent(dto.EventC2CMessageCreate) == 0 {
		t.Error("intents missing C2C_MESSAGE_CREATE")
	}
	if intents&dto.EventToIntent(dto.EventGroupAtMessageCreate) == 0 {
		t.Error("intents missing GROUP_AT_MESSAGE_CREATE")
	}
	if intents&dto.EventToIntent(dto.EventGroupAtMessageCreate) == 0 {
		t.Error("intents missing GROUP_AT_MESSAGE_CREATE")
	}
}
