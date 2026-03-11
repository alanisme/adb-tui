package adb

import (
	"testing"
)

func TestSettingNamespaceConstants(t *testing.T) {
	if NamespaceSystem != "system" {
		t.Fatalf("expected system, got %s", NamespaceSystem)
	}
	if NamespaceSecure != "secure" {
		t.Fatalf("expected secure, got %s", NamespaceSecure)
	}
	if NamespaceGlobal != "global" {
		t.Fatalf("expected global, got %s", NamespaceGlobal)
	}
}

func TestParseSettingsOutput(t *testing.T) {
	output := "screen_brightness=128\nfont_scale=1.0\nscreen_off_timeout=60000\n"
	settings := parseSettingsOutput(output)

	if len(settings) != 3 {
		t.Fatalf("expected 3 settings, got %d", len(settings))
	}
	if settings["screen_brightness"] != "128" {
		t.Fatalf("expected 128, got %s", settings["screen_brightness"])
	}
	if settings["font_scale"] != "1.0" {
		t.Fatalf("expected 1.0, got %s", settings["font_scale"])
	}
	if settings["screen_off_timeout"] != "60000" {
		t.Fatalf("expected 60000, got %s", settings["screen_off_timeout"])
	}
}

func TestParseSettingsOutput_Empty(t *testing.T) {
	settings := parseSettingsOutput("")
	if len(settings) != 0 {
		t.Fatalf("expected 0 settings, got %d", len(settings))
	}
}

func TestParseSettingsOutput_BlankLines(t *testing.T) {
	output := "key1=val1\n\n\nkey2=val2\n"
	settings := parseSettingsOutput(output)
	if len(settings) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(settings))
	}
}

func TestParseSettingsOutput_NoEqualsSign(t *testing.T) {
	output := "key1=val1\nbadline\nkey2=val2\n"
	settings := parseSettingsOutput(output)
	if len(settings) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(settings))
	}
}

func TestParseSettingsOutput_ValueWithEquals(t *testing.T) {
	output := "key1=val=ue\n"
	settings := parseSettingsOutput(output)
	if settings["key1"] != "val=ue" {
		t.Fatalf("expected val=ue, got %s", settings["key1"])
	}
}

func TestParseSettingsOutput_EmptyValue(t *testing.T) {
	output := "key1=\n"
	settings := parseSettingsOutput(output)
	if settings["key1"] != "" {
		t.Fatalf("expected empty value, got %s", settings["key1"])
	}
}
