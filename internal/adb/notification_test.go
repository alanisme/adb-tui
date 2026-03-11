package adb

import "testing"

func TestParseNotifications(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []NotificationInfo
	}{
		{
			name: "standard format",
			input: `  NotificationRecord(0xabc123: pkg=com.example.app user=UserHandle{0} importance=3 key=0|com.example.app|1001|null|10042):
    pkg=com.example.app
    opPkg=com.example.app
    extras={
      android.title=String (My Title)
      android.text=String (Hello World)
      android.subText=null
    }
    postTime=1234567890`,
			want: []NotificationInfo{
				{
					Key:     "0xabc123: pkg=com.example.app user=UserHandle{0} importance=3 key=0|com.example.app|1001|null|10042",
					Package: "com.example.app",
					Title:   "My Title",
					Text:    "Hello World",
					Time:    "1234567890",
				},
			},
		},
		{
			name: "text with parentheses",
			input: `  NotificationRecord(0xdef456:):
    pkg=com.test
    extras={
      android.title=String (Alert (urgent))
      android.text=String (Check this out :))
    }`,
			want: []NotificationInfo{
				{
					Key:     "0xdef456:",
					Package: "com.test",
					Title:   "Alert (urgent)",
					Text:    "Check this out :)",
				},
			},
		},
		{
			name: "multiple notifications",
			input: `  NotificationRecord(0x001:):
    pkg=com.app1
    extras={
      android.title=String (First)
    }
  NotificationRecord(0x002:):
    pkg=com.app2
    extras={
      android.title=String (Second)
      android.text=String (Details)
    }`,
			want: []NotificationInfo{
				{Key: "0x001:", Package: "com.app1", Title: "First"},
				{Key: "0x002:", Package: "com.app2", Title: "Second", Text: "Details"},
			},
		},
		{
			name:  "empty output",
			input: "",
			want:  nil,
		},
		{
			name: "no extras section",
			input: `  NotificationRecord(0x789:):
    pkg=com.bare
    postTime=999`,
			want: []NotificationInfo{
				{Key: "0x789:", Package: "com.bare", Time: "999"},
			},
		},
		{
			name: "inline extras with parens in text",
			input: `  NotificationRecord(0xabc:):
    pkg=com.test
    extras={android.title=String (Hello :)), android.text=String (World (2))}`,
			want: []NotificationInfo{
				{Key: "0xabc:", Package: "com.test", Title: "Hello :)", Text: "World (2)"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNotifications(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d notifications, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Package != tt.want[i].Package {
					t.Errorf("[%d] Package = %q, want %q", i, got[i].Package, tt.want[i].Package)
				}
				if got[i].Title != tt.want[i].Title {
					t.Errorf("[%d] Title = %q, want %q", i, got[i].Title, tt.want[i].Title)
				}
				if got[i].Text != tt.want[i].Text {
					t.Errorf("[%d] Text = %q, want %q", i, got[i].Text, tt.want[i].Text)
				}
				if tt.want[i].Time != "" && got[i].Time != tt.want[i].Time {
					t.Errorf("[%d] Time = %q, want %q", i, got[i].Time, tt.want[i].Time)
				}
			}
		})
	}
}

func TestExtractNotifValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"String (Hello)", "Hello"},
		{"String (Hello (world))", "Hello (world)"},
		{"String ()", ""},
		{"plain value", "plain value"},
		{"null", "null"},
	}
	for _, tt := range tests {
		got := extractNotifValue(tt.input)
		if got != tt.want {
			t.Errorf("extractNotifValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
