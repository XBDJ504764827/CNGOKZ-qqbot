package rule

import (
	"testing"
	"time"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

func newTestRules() *Rules {
	return New(5 * time.Minute)
}

func TestShouldNotify_EnabledEvents(t *testing.T) {
	rules := newTestRules()

	cases := []struct {
		name      string
		eventType string
		level     string
		want      bool
	}{
		{"server offline critical", event.EventServerOffline, event.LevelCritical, true},
		{"server offline info", event.EventServerOffline, event.LevelInfo, true},
		{"server online", event.EventServerOnline, event.LevelInfo, true},
		{"system warning warning", event.EventSystemWarning, event.LevelWarning, true},
		{"forum report warning", event.EventForumReportCreated, event.LevelWarning, true},
		{"whitelist request warning", event.EventWhitelistRequestCreated, event.LevelWarning, true},
		{"whitelist request info (too low)", event.EventWhitelistRequestCreated, event.LevelInfo, false},
		{"admin action disabled", event.EventAdminAction, event.LevelCritical, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, _ := rules.ShouldNotify(event.Event{EventType: tc.eventType, Level: tc.level})
			if ok != tc.want {
				t.Errorf("ShouldNotify(%s, %s) = %v, want %v", tc.eventType, tc.level, ok, tc.want)
			}
		})
	}
}

func TestShouldNotify_UnknownEvent(t *testing.T) {
	rules := newTestRules()
	ok, _ := rules.ShouldNotify(event.Event{EventType: "MESSAGE_RECEIVED", Level: event.LevelCritical})
	if ok {
		t.Error("unknown event type should not notify")
	}
}

func TestShouldNotify_LevelTooLow(t *testing.T) {
	rules := newTestRules()
	// SYSTEM_WARNING 规则要求 >= warning，info 级别不通知
	ok, _ := rules.ShouldNotify(event.Event{EventType: event.EventSystemWarning, Level: event.LevelInfo})
	if ok {
		t.Error("info level should not trigger SYSTEM_WARNING notification")
	}
}

func TestRule_CooldownConfigured(t *testing.T) {
	rules := newTestRules()
	_, r := rules.ShouldNotify(event.Event{EventType: event.EventServerOffline, Level: event.LevelCritical})
	if r.Cooldown != 5*time.Minute {
		t.Errorf("Cooldown = %v, want %v", r.Cooldown, 5*time.Minute)
	}
}
