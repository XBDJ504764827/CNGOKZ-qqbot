package message

import (
	"context"
	"errors"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"github.com/tencent-connect/botgo/openapi/options"
	"go.uber.org/zap"
)

// fakeMessageAPI 最小发送接口的测试替身。
type fakeMessageAPI struct {
	postMessage      func(ctx context.Context, channelID string, msg *dto.MessageToCreate, opt ...options.Option) (*dto.Message, error)
	postGroupMessage func(ctx context.Context, groupID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error)
	postC2CMessage   func(ctx context.Context, userID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error)
}

func (f *fakeMessageAPI) PostMessage(ctx context.Context, channelID string, msg *dto.MessageToCreate, opt ...options.Option) (*dto.Message, error) {
	return f.postMessage(ctx, channelID, msg, opt...)
}

func (f *fakeMessageAPI) PostGroupMessage(ctx context.Context, groupID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error) {
	return f.postGroupMessage(ctx, groupID, msg, opt...)
}

func (f *fakeMessageAPI) PostC2CMessage(ctx context.Context, userID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error) {
	return f.postC2CMessage(ctx, userID, msg, opt...)
}

func TestSendChannelMessage(t *testing.T) {
	var gotChannelID string
	var gotContent string

	api := &fakeMessageAPI{
		postMessage: func(_ context.Context, channelID string, msg *dto.MessageToCreate, _ ...options.Option) (*dto.Message, error) {
			gotChannelID, gotContent = channelID, msg.Content
			return &dto.Message{ID: "msg-1"}, nil
		},
	}
	s := NewSender(api, zap.NewNop())

	got, err := s.SendChannelMessage(context.Background(), "channel-1", "hello")
	if err != nil {
		t.Fatalf("SendChannelMessage() error = %v", err)
	}
	if got.ID != "msg-1" {
		t.Errorf("got message id = %q, want %q", got.ID, "msg-1")
	}
	if gotChannelID != "channel-1" || gotContent != "hello" {
		t.Errorf("postMessage args = (%q, %q), want (%q, %q)", gotChannelID, gotContent, "channel-1", "hello")
	}
}

func TestSendGroupMessage(t *testing.T) {
	var gotGroupID string
	var gotContent string

	api := &fakeMessageAPI{
		postGroupMessage: func(_ context.Context, groupID string, msg dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
			gotGroupID, gotContent = groupID, msg.(*dto.MessageToCreate).Content
			return &dto.Message{ID: "msg-2"}, nil
		},
	}
	s := NewSender(api, zap.NewNop())

	if _, err := s.SendGroupMessage(context.Background(), "group-1", "notice"); err != nil {
		t.Fatalf("SendGroupMessage() error = %v", err)
	}
	if gotGroupID != "group-1" || gotContent != "notice" {
		t.Errorf("postGroupMessage args = (%q, %q), want (%q, %q)", gotGroupID, gotContent, "group-1", "notice")
	}
}

func TestSendC2CMessage(t *testing.T) {
	var gotUserID string
	var gotContent string

	api := &fakeMessageAPI{
		postC2CMessage: func(_ context.Context, userID string, msg dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
			gotUserID, gotContent = userID, msg.(*dto.MessageToCreate).Content
			return &dto.Message{ID: "msg-3"}, nil
		},
	}
	s := NewSender(api, zap.NewNop())

	if _, err := s.SendC2CMessage(context.Background(), "user-1", "admin alert"); err != nil {
		t.Fatalf("SendC2CMessage() error = %v", err)
	}
	if gotUserID != "user-1" || gotContent != "admin alert" {
		t.Errorf("postC2CMessage args = (%q, %q), want (%q, %q)", gotUserID, gotContent, "user-1", "admin alert")
	}
}

func TestSendC2CMessageWithKeyboard(t *testing.T) {
	var gotMsg *dto.MessageToCreate

	api := &fakeMessageAPI{
		postC2CMessage: func(_ context.Context, _ string, msg dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
			gotMsg = msg.(*dto.MessageToCreate)
			return &dto.Message{ID: "msg-4"}, nil
		},
	}
	s := NewSender(api, zap.NewNop())

	kb := &keyboard.CustomKeyboard{Rows: []*keyboard.Row{{Buttons: []*keyboard.Button{
		{ID: "approve:wl-1", RenderData: &keyboard.RenderData{Label: "通过"},
			Action: &keyboard.Action{Type: keyboard.ActionTypeCallback, Data: "approve:wl-1"}},
	}}}}

	if _, err := s.SendC2CMessageWithKeyboard(context.Background(), "user-1", "hello", kb); err != nil {
		t.Fatalf("SendC2CMessageWithKeyboard() error = %v", err)
	}
	if gotMsg.MsgType != dto.MarkdownMsg {
		t.Errorf("msg_type = %d, want %d (markdown)：纯文本携带 keyboard 会被服务端静默丢弃", gotMsg.MsgType, dto.MarkdownMsg)
	}
	if gotMsg.Markdown == nil || gotMsg.Markdown.Content != "hello" {
		t.Errorf("markdown = %+v, want content %q", gotMsg.Markdown, "hello")
	}
	if gotMsg.Keyboard == nil || gotMsg.Keyboard.Content != kb {
		t.Errorf("keyboard = %+v, want %+v", gotMsg.Keyboard, kb)
	}
}

func TestSendC2CMessageWithKeyboardFallback(t *testing.T) {
	var calls int

	api := &fakeMessageAPI{
		postC2CMessage: func(_ context.Context, _ string, msg dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
			calls++
			m := msg.(*dto.MessageToCreate)
			if m.MsgType == dto.MarkdownMsg {
				return nil, errors.New("markdown not allowed")
			}
			return &dto.Message{ID: "msg-5"}, nil
		},
	}
	s := NewSender(api, zap.NewNop())

	kb := &keyboard.CustomKeyboard{Rows: []*keyboard.Row{{Buttons: []*keyboard.Button{
		{ID: "approve:wl-1", RenderData: &keyboard.RenderData{Label: "通过"},
			Action: &keyboard.Action{Type: keyboard.ActionTypeCallback, Data: "approve:wl-1"}},
	}}}}

	if _, err := s.SendC2CMessageWithKeyboard(context.Background(), "user-1", "hello", kb); err != nil {
		t.Fatalf("expected fallback to plain text, got error = %v", err)
	}
	if calls != 2 {
		t.Errorf("postC2CMessage calls = %d, want 2 (markdown 失败后降级纯文本)", calls)
	}
}

func TestSendValidation(t *testing.T) {
	s := NewSender(&fakeMessageAPI{}, zap.NewNop())

	cases := []struct {
		name    string
		target  string
		content string
	}{
		{"empty target", "", "hello"},
		{"empty content", "channel-1", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.SendChannelMessage(context.Background(), tc.target, tc.content); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestSendAPIFailure(t *testing.T) {
	wantErr := errors.New("api unavailable")
	api := &fakeMessageAPI{
		postMessage: func(_ context.Context, _ string, _ *dto.MessageToCreate, _ ...options.Option) (*dto.Message, error) {
			return nil, wantErr
		},
	}
	s := NewSender(api, zap.NewNop())

	if _, err := s.SendChannelMessage(context.Background(), "channel-1", "hello"); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
