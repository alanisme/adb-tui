package adb

import (
	"fmt"
	"strings"
	"testing"
)

// buildMonkeyArgs mirrors the arg-building logic in RunMonkey for testing.
func buildMonkeyArgs(pkg string, events int, options MonkeyOptions) []string {
	args := []string{"monkey", "-p", pkg}
	if options.Seed != 0 {
		args = append(args, "-s", fmt.Sprintf("%d", options.Seed))
	}
	if options.Throttle > 0 {
		args = append(args, "--throttle", fmt.Sprintf("%d", options.Throttle))
	}
	if options.IgnoreCrashes {
		args = append(args, "--ignore-crashes")
	}
	if options.IgnoreTimeouts {
		args = append(args, "--ignore-timeouts")
	}
	if options.IgnoreSecurityExceptions {
		args = append(args, "--ignore-security-exceptions")
	}
	args = append(args, "-v", fmt.Sprintf("%d", events))
	return args
}

// buildInstrumentArgs mirrors the arg-building logic in RunInstrumentation for testing.
func buildInstrumentArgs(component string, options InstrumentOptions) []string {
	args := []string{"am", "instrument"}
	if options.RawOutput {
		args = append(args, "-r")
	}
	if options.NoWindowAnimation {
		args = append(args, "--no-window-animation")
	}
	if options.Runner != "" {
		args = append(args, "-e", "class", options.Runner)
	}
	for k, v := range options.Arguments {
		args = append(args, "-e", k, v)
	}
	args = append(args, "-w", component)
	return args
}

func TestBuildMonkeyArgs_Defaults(t *testing.T) {
	args := buildMonkeyArgs("com.example.app", 500, MonkeyOptions{})
	cmd := strings.Join(args, " ")
	if cmd != "monkey -p com.example.app -v 500" {
		t.Fatalf("unexpected command: %s", cmd)
	}
}

func TestBuildMonkeyArgs_AllOptions(t *testing.T) {
	opts := MonkeyOptions{
		Seed:                     42,
		Throttle:                 300,
		IgnoreCrashes:            true,
		IgnoreTimeouts:           true,
		IgnoreSecurityExceptions: true,
	}
	args := buildMonkeyArgs("com.test", 1000, opts)
	cmd := strings.Join(args, " ")

	if !strings.Contains(cmd, "-p com.test") {
		t.Fatalf("missing package: %s", cmd)
	}
	if !strings.Contains(cmd, "-s 42") {
		t.Fatalf("missing seed: %s", cmd)
	}
	if !strings.Contains(cmd, "--throttle 300") {
		t.Fatalf("missing throttle: %s", cmd)
	}
	if !strings.Contains(cmd, "--ignore-crashes") {
		t.Fatalf("missing ignore-crashes: %s", cmd)
	}
	if !strings.Contains(cmd, "--ignore-timeouts") {
		t.Fatalf("missing ignore-timeouts: %s", cmd)
	}
	if !strings.Contains(cmd, "--ignore-security-exceptions") {
		t.Fatalf("missing ignore-security-exceptions: %s", cmd)
	}
	if !strings.HasSuffix(cmd, "-v 1000") {
		t.Fatalf("expected events at end: %s", cmd)
	}
}

func TestBuildMonkeyArgs_SeedOnly(t *testing.T) {
	args := buildMonkeyArgs("com.app", 100, MonkeyOptions{Seed: 12345})
	cmd := strings.Join(args, " ")
	if !strings.Contains(cmd, "-s 12345") {
		t.Fatalf("missing seed: %s", cmd)
	}
	if strings.Contains(cmd, "--throttle") {
		t.Fatalf("unexpected throttle: %s", cmd)
	}
}

func TestBuildInstrumentArgs_Defaults(t *testing.T) {
	args := buildInstrumentArgs("com.test/.TestRunner", InstrumentOptions{})
	cmd := strings.Join(args, " ")
	if cmd != "am instrument -w com.test/.TestRunner" {
		t.Fatalf("unexpected command: %s", cmd)
	}
}

func TestBuildInstrumentArgs_RawOutput(t *testing.T) {
	args := buildInstrumentArgs("com.test/.Runner", InstrumentOptions{RawOutput: true})
	cmd := strings.Join(args, " ")
	if !strings.Contains(cmd, " -r") {
		t.Fatalf("missing -r flag: %s", cmd)
	}
}

func TestBuildInstrumentArgs_NoWindowAnimation(t *testing.T) {
	args := buildInstrumentArgs("com.test/.Runner", InstrumentOptions{NoWindowAnimation: true})
	cmd := strings.Join(args, " ")
	if !strings.Contains(cmd, "--no-window-animation") {
		t.Fatalf("missing no-window-animation: %s", cmd)
	}
}

func TestBuildInstrumentArgs_WithRunner(t *testing.T) {
	opts := InstrumentOptions{Runner: "com.test.MyTest#testMethod"}
	args := buildInstrumentArgs("com.test/.Runner", opts)
	cmd := strings.Join(args, " ")
	if !strings.Contains(cmd, "-e class com.test.MyTest#testMethod") {
		t.Fatalf("missing runner class: %s", cmd)
	}
}

func TestBuildInstrumentArgs_WithComponent(t *testing.T) {
	args := buildInstrumentArgs("com.example.test/androidx.test.runner.AndroidJUnitRunner", InstrumentOptions{})
	cmd := strings.Join(args, " ")
	if !strings.HasSuffix(cmd, "-w com.example.test/androidx.test.runner.AndroidJUnitRunner") {
		t.Fatalf("missing component at end: %s", cmd)
	}
}
