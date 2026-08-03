package event

import "testing"

func TestEvent_Validate(t *testing.T) {
	valid := Event{Source: "LumiForum", EventType: EventForumReportCreated, Level: LevelWarning}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() valid event error = %v", err)
	}

	// 空 level 合法（Normalize 补全）
	noLevel := Event{Source: "LumiForum", EventType: EventForumReportCreated}
	if err := noLevel.Validate(); err != nil {
		t.Fatalf("Validate() event without level error = %v", err)
	}
}

func TestEvent_ValidateMissingFields(t *testing.T) {
	cases := []struct {
		name  string
		event Event
	}{
		{"missing source", Event{EventType: "X"}},
		{"missing event_type", Event{Source: "LumiForum"}},
		{"empty source", Event{Source: "  ", EventType: "X"}},
		{"empty event_type", Event{Source: "LumiForum", EventType: "  "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.event.Validate(); err == nil {
				t.Fatal("Validate() expected error, got nil")
			}
		})
	}
}

func TestEvent_ValidateInvalidLevel(t *testing.T) {
	ev := Event{Source: "LumiForum", EventType: "X", Level: "fatal"}
	if err := ev.Validate(); err == nil {
		t.Fatal("Validate() expected error for invalid level, got nil")
	}
}

// TestEvent_Normalize 验证字段补全：ID / Level / Timestamp / Data。
func TestEvent_Normalize(t *testing.T) {
	ev := Event{Source: "LumiForum", EventType: "X"}
	ev.Normalize()

	if ev.ID == "" {
		t.Error("Normalize() should generate ID")
	}
	if ev.Level != LevelInfo {
		t.Errorf("Level = %q, want %q", ev.Level, LevelInfo)
	}
	if ev.Timestamp.IsZero() {
		t.Error("Normalize() should set Timestamp")
	}
	if ev.Data == nil {
		t.Error("Normalize() should initialize Data map")
	}
}

// TestEvent_NormalizeKeepProvided 验证已提供的字段不被覆盖。
func TestEvent_NormalizeKeepProvided(t *testing.T) {
	ev := Event{ID: "keep-me", Level: LevelCritical}
	ev.Normalize()
	if ev.ID != "keep-me" || ev.Level != LevelCritical {
		t.Fatalf("Normalize() overwrote provided fields: %+v", ev)
	}
}
