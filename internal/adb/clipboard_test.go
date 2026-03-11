package adb

import "testing"

func TestParseClipboardDump(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "T: format",
			input: `Current user: 0
Primary clip:
  text/plain
  T:Hello World`,
			want: "Hello World",
		},
		{
			name: "mText format",
			input: `  mPrimaryClip=ClipData { text/plain
    mText=Clipboard content here
  }`,
			want: "Clipboard content here",
		},
		{
			name: "plain text on own line",
			input: `Primary clip {
  text/plain
  Some plain text
}`,
			want: "Some plain text",
		},
		{
			name:  "empty clipboard",
			input: `Current user: 0`,
			want:  "",
		},
		{
			name: "no text after header",
			input: `Primary clip:
  {empty}`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseClipboardDump(tt.input)
			if got != tt.want {
				t.Errorf("parseClipboardDump() = %q, want %q", got, tt.want)
			}
		})
	}
}
