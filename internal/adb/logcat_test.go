package adb

import (
	"testing"
)

func TestParseLogLine_Standard(t *testing.T) {
	line := "01-15 10:30:45.123  1234  5678 I ActivityManager: Start proc com.example"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.PID != "1234" {
		t.Fatalf("expected pid 1234, got %s", entry.PID)
	}
	if entry.TID != "5678" {
		t.Fatalf("expected tid 5678, got %s", entry.TID)
	}
	if entry.Level != LogInfo {
		t.Fatalf("expected I, got %s", entry.Level)
	}
	if entry.Tag != "ActivityManager" {
		t.Fatalf("expected ActivityManager, got %s", entry.Tag)
	}
	if entry.Message != "Start proc com.example" {
		t.Fatalf("unexpected message: %s", entry.Message)
	}
}

func TestParseLogLine_Warning(t *testing.T) {
	line := "03-20 08:00:01.456  999  999 W System.err: something bad"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Level != LogWarn {
		t.Fatalf("expected W, got %s", entry.Level)
	}
	if entry.Tag != "System.err" {
		t.Fatalf("expected System.err, got %s", entry.Tag)
	}
}

func TestParseLogLine_Error(t *testing.T) {
	line := "12-31 23:59:59.999  100  200 E CrashReporter: fatal error occurred"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Level != LogError {
		t.Fatalf("expected E, got %s", entry.Level)
	}
}

func TestParseLogLine_Debug(t *testing.T) {
	line := "06-01 12:00:00.000  500  600 D MyApp: debug message here"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Level != LogDebug {
		t.Fatalf("expected D, got %s", entry.Level)
	}
}

func TestParseLogLine_Verbose(t *testing.T) {
	line := "06-01 12:00:00.000  500  600 V MyApp: verbose"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Level != LogVerbose {
		t.Fatalf("expected V, got %s", entry.Level)
	}
}

func TestParseLogLine_Fatal(t *testing.T) {
	line := "06-01 12:00:00.000  500  600 F MyApp: fatal"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Level != LogFatal {
		t.Fatalf("expected F, got %s", entry.Level)
	}
}

func TestParseLogLine_Empty(t *testing.T) {
	_, ok := parseLogLine("")
	if ok {
		t.Fatal("expected not ok for empty")
	}
}

func TestParseLogLine_Separator(t *testing.T) {
	_, ok := parseLogLine("--------- beginning of main")
	if ok {
		t.Fatal("expected not ok for separator")
	}
}

func TestParseLogLine_TooFewFields(t *testing.T) {
	_, ok := parseLogLine("01-15 10:30:45.123")
	if ok {
		t.Fatal("expected not ok for too few fields")
	}
}

func TestParseLogLine_NoMessage(t *testing.T) {
	line := "01-15 10:30:45.123  1234  5678 I Tag:"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Message != "" {
		t.Fatalf("expected empty message, got %s", entry.Message)
	}
}

func TestParseLogLine_TimestampParsing(t *testing.T) {
	line := "03-15 14:30:45.123  1000  2000 I Test: msg"
	entry, ok := parseLogLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Timestamp.Month() != 3 {
		t.Fatalf("expected month 3, got %d", entry.Timestamp.Month())
	}
	if entry.Timestamp.Day() != 15 {
		t.Fatalf("expected day 15, got %d", entry.Timestamp.Day())
	}
	if entry.Timestamp.Hour() != 14 {
		t.Fatalf("expected hour 14, got %d", entry.Timestamp.Hour())
	}
}

func TestBuildLogcatArgs_Dump(t *testing.T) {
	c := NewClientWithPath("/usr/bin/adb")
	opts := LogcatOptions{
		Format: "threadtime",
		Buffer: "main",
		Since:  "2024-01-01",
		Count:  100,
		Filter: "ActivityManager:I",
	}
	args := c.buildLogcatArgs("device123", opts, true)

	expected := []string{"-s", "device123", "logcat", "-d", "-v", "threadtime", "-b", "main", "-T", "2024-01-01", "-t", "100", "ActivityManager:I"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, e := range expected {
		if args[i] != e {
			t.Fatalf("arg[%d]: expected %s, got %s", i, e, args[i])
		}
	}
}

func TestBuildLogcatArgs_Stream(t *testing.T) {
	c := NewClientWithPath("/usr/bin/adb")
	opts := LogcatOptions{}
	args := c.buildLogcatArgs("", opts, false)

	expected := []string{"logcat"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
}

func TestBuildLogcatArgs_NoSerial(t *testing.T) {
	c := NewClientWithPath("/usr/bin/adb")
	opts := LogcatOptions{Format: "brief"}
	args := c.buildLogcatArgs("", opts, true)

	if args[0] != "logcat" {
		t.Fatalf("expected logcat as first arg, got %s", args[0])
	}
}

func TestLogLevelConstants(t *testing.T) {
	if LogVerbose != "V" {
		t.Fatal("unexpected LogVerbose")
	}
	if LogDebug != "D" {
		t.Fatal("unexpected LogDebug")
	}
	if LogInfo != "I" {
		t.Fatal("unexpected LogInfo")
	}
	if LogWarn != "W" {
		t.Fatal("unexpected LogWarn")
	}
	if LogError != "E" {
		t.Fatal("unexpected LogError")
	}
	if LogFatal != "F" {
		t.Fatal("unexpected LogFatal")
	}
}
