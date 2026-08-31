package message

import (
	"context"
	"errors"
	"testing"

	"github.com/tencent-connect/botgo/dto"
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
