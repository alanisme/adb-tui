package adb

import (
	"testing"
)

func TestParseBatteryStatus(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"1", "Unknown"},
		{"2", "Charging"},
		{"3", "Discharging"},
		{"4", "Not charging"},
		{"5", "Full"},
		{"99", "99"},
		{"", ""},
	}
	for _, tc := range cases {
		got := parseBatteryStatus(tc.input)
		if got != tc.expected {
			t.Fatalf("parseBatteryStatus(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestParseBatteryHealth(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"1", "Unknown"},
		{"2", "Good"},
		{"3", "Overheat"},
		{"4", "Dead"},
		{"5", "Over voltage"},
		{"6", "Failure"},
		{"7", "Cold"},
		{"0", "0"},
		{"abc", "abc"},
	}
	for _, tc := range cases {
		got := parseBatteryHealth(tc.input)
		if got != tc.expected {
			t.Fatalf("parseBatteryHealth(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestParseBatteryOutput(t *testing.T) {
	output := `Current Battery Service state:
  AC powered: false
  USB powered: true
  status: 2
  health: 2
  level: 85
  temperature: 270
  voltage: 4200
  technology: Li-ion`

	info := parseBatteryOutput(output)
	if info.Level != 85 {
		t.Fatalf("expected level 85, got %d", info.Level)
	}
	if info.Temperature != 270 {
		t.Fatalf("expected temp 270, got %d", info.Temperature)
	}
	if info.Voltage != 4200 {
		t.Fatalf("expected voltage 4200, got %d", info.Voltage)
	}
	if info.Status != "Charging" {
		t.Fatalf("expected Charging, got %s", info.Status)
	}
	if info.Health != "Good" {
		t.Fatalf("expected Good, got %s", info.Health)
	}
}

func TestParseBatteryOutput_Empty(t *testing.T) {
	info := parseBatteryOutput("")
	if info.Level != 0 || info.Status != "" {
		t.Fatal("expected zero values for empty input")
	}
}

func TestParseDfOutput(t *testing.T) {
	output := `Filesystem     Size  Used Avail Use% Mounted on
/dev/block/dm-0  4.8G  3.2G  1.5G  69% /
tmpfs            1.9G  580K  1.9G   1% /dev
/dev/block/sda1  24G   18G   5.2G  78% /data`

	disks := parseDfOutput(output)
	if len(disks) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(disks))
	}
	if disks[0].Filesystem != "/dev/block/dm-0" {
		t.Fatalf("expected /dev/block/dm-0, got %s", disks[0].Filesystem)
	}
	if disks[0].Size != "4.8G" {
		t.Fatalf("expected 4.8G, got %s", disks[0].Size)
	}
	if disks[0].MountPoint != "/" {
		t.Fatalf("expected /, got %s", disks[0].MountPoint)
	}
	if disks[2].UsePercent != "78%" {
		t.Fatalf("expected 78%%, got %s", disks[2].UsePercent)
	}
}

func TestParseDfOutput_Empty(t *testing.T) {
	disks := parseDfOutput("")
	if len(disks) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(disks))
	}
}

func TestParseDfOutput_HeaderOnly(t *testing.T) {
	output := "Filesystem     Size  Used Avail Use% Mounted on\n"
	disks := parseDfOutput(output)
	if len(disks) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(disks))
	}
}

func TestParseDfOutput_TooFewFields(t *testing.T) {
	output := "/dev/block 4.8G 3.2G\n"
	disks := parseDfOutput(output)
	if len(disks) != 0 {
		t.Fatalf("expected 0 entries for too-few fields, got %d", len(disks))
	}
}

func TestParseDumpsysList(t *testing.T) {
	output := "activity\nbattery\nwindow\nwifi\n"
	services := parseDumpsysList(output)
	if len(services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(services))
	}
	if services[0] != "activity" {
		t.Fatalf("expected activity, got %s", services[0])
	}
	if services[3] != "wifi" {
		t.Fatalf("expected wifi, got %s", services[3])
	}
}

func TestParseDumpsysList_Empty(t *testing.T) {
	services := parseDumpsysList("")
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}
}

func TestParseDumpsysList_BlankLines(t *testing.T) {
	output := "battery\n\n\nwifi\n"
	services := parseDumpsysList(output)
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
}

func TestParseDumpsysList_WithWhitespace(t *testing.T) {
	output := "  battery  \n  wifi  \n"
	services := parseDumpsysList(output)
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0] != "battery" {
		t.Fatalf("expected battery, got %s", services[0])
	}
}
