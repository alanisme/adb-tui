package adb

import (
	"testing"
)

func TestParseProcessLine_Full(t *testing.T) {
	line := "1234 root 12.5 3.2 com.example.app"
	p := parseProcessLine(line)
	if p == nil {
		t.Fatal("expected non-nil process")
	}
	if p.PID != 1234 {
		t.Fatalf("expected pid 1234, got %d", p.PID)
	}
	if p.User != "root" {
		t.Fatalf("expected root, got %s", p.User)
	}
	if p.CPU != 12.5 {
		t.Fatalf("expected cpu 12.5, got %f", p.CPU)
	}
	if p.MEM != 3.2 {
		t.Fatalf("expected mem 3.2, got %f", p.MEM)
	}
	if p.Name != "com.example.app" {
		t.Fatalf("expected com.example.app, got %s", p.Name)
	}
}

func TestParseProcessLine_WithPercentSuffix(t *testing.T) {
	line := "567 u0_a123 5.0% 1.0% com.test"
	p := parseProcessLine(line)
	if p == nil {
		t.Fatal("expected non-nil process")
	}
	if p.CPU != 5.0 {
		t.Fatalf("expected cpu 5.0, got %f", p.CPU)
	}
	if p.MEM != 1.0 {
		t.Fatalf("expected mem 1.0, got %f", p.MEM)
	}
}

func TestParseProcessLine_Minimal(t *testing.T) {
	line := "100 root"
	p := parseProcessLine(line)
	if p == nil {
		t.Fatal("expected non-nil process")
	}
	if p.PID != 100 {
		t.Fatalf("expected pid 100, got %d", p.PID)
	}
	if p.User != "root" {
		t.Fatalf("expected root, got %s", p.User)
	}
	if p.Name != "root" {
		t.Fatalf("expected name root, got %s", p.Name)
	}
}

func TestParseProcessLine_SingleField(t *testing.T) {
	p := parseProcessLine("abc")
	if p != nil {
		t.Fatal("expected nil for single field")
	}
}

func TestParseProcessLine_InvalidPID(t *testing.T) {
	p := parseProcessLine("abc root 1.0 2.0 test")
	if p != nil {
		t.Fatal("expected nil for invalid pid")
	}
}

func TestParseProcessLine_Empty(t *testing.T) {
	p := parseProcessLine("")
	if p != nil {
		t.Fatal("expected nil for empty line")
	}
}

func TestParseMemInfoOutput(t *testing.T) {
	output := `MemTotal:        3940168 kB
MemFree:          157384 kB
MemAvailable:    1823456 kB
Buffers:           52348 kB
Cached:          1234567 kB`

	info := parseMemInfoOutput(output)
	if info.Total != 3940168 {
		t.Fatalf("expected total 3940168, got %d", info.Total)
	}
	if info.Free != 157384 {
		t.Fatalf("expected free 157384, got %d", info.Free)
	}
	if info.Available != 1823456 {
		t.Fatalf("expected available 1823456, got %d", info.Available)
	}
}

func TestParseMemInfoOutput_Empty(t *testing.T) {
	info := parseMemInfoOutput("")
	if info.Total != 0 || info.Free != 0 || info.Available != 0 {
		t.Fatal("expected all zeros for empty input")
	}
}

func TestParseMemInfoOutput_Partial(t *testing.T) {
	output := "MemTotal:        2048000 kB\n"
	info := parseMemInfoOutput(output)
	if info.Total != 2048000 {
		t.Fatalf("expected 2048000, got %d", info.Total)
	}
	if info.Free != 0 {
		t.Fatalf("expected 0 free, got %d", info.Free)
	}
}

func TestParseTopOutput(t *testing.T) {
	output := `Tasks: 320 total,   1 running, 319 sleeping
Mem:   3940168k total,  3782784k used,   157384k free
Swap:        0k total,        0k used,        0k free

  PID USER     %CPU %MEM NAME
 1234 root     25.0  5.0 com.process.a extra_field
  567 system    3.5  1.2 com.process.b
  890 u0_a12    0.1  0.5 com.process.c`

	procs := parseTopOutput(output, 0)
	if len(procs) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(procs))
	}
	if procs[0].PID != 1234 {
		t.Fatalf("expected pid 1234, got %d", procs[0].PID)
	}
	if procs[0].CPU != 25.0 {
		t.Fatalf("expected cpu 25.0, got %f", procs[0].CPU)
	}
	if procs[0].Name != "extra_field" {
		t.Fatalf("expected last field as name, got %s", procs[0].Name)
	}
	if procs[1].User != "system" {
		t.Fatalf("expected system, got %s", procs[1].User)
	}
}

func TestParseTopOutput_WithLimit(t *testing.T) {
	output := `  PID USER     %CPU %MEM NAME
 1 root     10.0  2.0 init
 2 root      5.0  1.0 kthreadd
 3 root      3.0  0.5 ksoftirqd`

	procs := parseTopOutput(output, 2)
	if len(procs) != 2 {
		t.Fatalf("expected 2 processes, got %d", len(procs))
	}
}

func TestParseTopOutput_Empty(t *testing.T) {
	procs := parseTopOutput("", 0)
	if len(procs) != 0 {
		t.Fatalf("expected 0 processes, got %d", len(procs))
	}
}

func TestParseTopOutput_NoHeader(t *testing.T) {
	output := "some random text\nanother line\n"
	procs := parseTopOutput(output, 0)
	if len(procs) != 0 {
		t.Fatalf("expected 0 processes, got %d", len(procs))
	}
}
