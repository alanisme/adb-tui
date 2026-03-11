package adb

import (
	"testing"
)

func TestParseProps(t *testing.T) {
	output := `[ro.product.model]: [Pixel 6]
[ro.product.brand]: [google]
[ro.build.version.release]: [14]
[ro.build.version.sdk]: [34]
[ro.product.cpu.abilist]: [arm64-v8a,armeabi-v7a,armeabi]`

	props := parsePropsOutput(output)

	if props["ro.product.model"] != "Pixel 6" {
		t.Fatalf("expected Pixel 6, got %s", props["ro.product.model"])
	}
	if props["ro.product.brand"] != "google" {
		t.Fatalf("expected google, got %s", props["ro.product.brand"])
	}
	if props["ro.build.version.release"] != "14" {
		t.Fatalf("expected 14, got %s", props["ro.build.version.release"])
	}
	if props["ro.build.version.sdk"] != "34" {
		t.Fatalf("expected 34, got %s", props["ro.build.version.sdk"])
	}
}

func TestParseProps_Empty(t *testing.T) {
	props := parsePropsOutput("")
	if len(props) != 0 {
		t.Fatalf("expected 0 props, got %d", len(props))
	}
}

func TestParseProps_MalformedLines(t *testing.T) {
	output := `not a prop
[valid.key]: [valid_value]
incomplete line
[another.key]: [another_value]`

	props := parsePropsOutput(output)
	if len(props) != 2 {
		t.Fatalf("expected 2 props, got %d", len(props))
	}
}

func TestParseProps_EmptyValue(t *testing.T) {
	output := `[ro.empty]: []`
	props := parsePropsOutput(output)
	if props["ro.empty"] != "" {
		t.Fatalf("expected empty value, got %s", props["ro.empty"])
	}
}

func TestParseProps_ValueWithBrackets(t *testing.T) {
	output := `[ro.test]: [value with ] bracket]`
	props := parsePropsOutput(output)
	if v, ok := props["ro.test"]; !ok {
		t.Fatal("expected key to exist")
	} else {
		_ = v
	}
}

func TestDeviceInfoStruct(t *testing.T) {
	info := DeviceInfo{
		AndroidVersion: "14",
		SDKVersion:     "34",
		Brand:          "google",
		Model:          "Pixel 6",
		ABIs:           []string{"arm64-v8a", "armeabi-v7a"},
	}
	if info.AndroidVersion != "14" {
		t.Fatal("unexpected android version")
	}
	if len(info.ABIs) != 2 {
		t.Fatalf("expected 2 ABIs, got %d", len(info.ABIs))
	}
}
