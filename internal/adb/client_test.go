package adb

import "testing"

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"", "''"},
		{"it's", "'it'\\''s'"},
		{"a'b'c", "'a'\\''b'\\''c'"},
		{"spaces here", "'spaces here'"},
		{"$(cmd)", "'$(cmd)'"},
		{"`cmd`", "'`cmd`'"},
		{"foo;bar", "'foo;bar'"},
		{"a\"b", "'a\"b'"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		v, lo, hi int
		want      int
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
		{0, -5, 5, 0},
	}

	for _, tt := range tests {
		got := clampInt(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}

func TestBatteryStatusName(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{1, "Unknown"},
		{2, "Charging"},
		{3, "Discharging"},
		{4, "Not charging"},
		{5, "Full"},
		{99, "Status 99"},
		{0, "Status 0"},
	}

	for _, tt := range tests {
		got := BatteryStatusName(tt.status)
		if got != tt.want {
			t.Errorf("BatteryStatusName(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestIsTCPDevice(t *testing.T) {
	tests := []struct {
		serial string
		want   bool
	}{
		{"192.168.1.100:5555", true},
		{"ABCD1234", false},
		{"emulator-5554", false},
		{"[::1]:5555", true},
		{"localhost:5555", true},
	}

	for _, tt := range tests {
		// isTCPDevice is in tui package, so test the logic directly
		got := len(tt.serial) > 0 && contains(tt.serial, ':')
		if got != tt.want {
			t.Errorf("isTCPDevice(%q) = %v, want %v", tt.serial, got, tt.want)
		}
	}
}

func contains(s string, c byte) bool {
	for i := range len(s) {
		if s[i] == c {
			return true
		}
	}
	return false
}
