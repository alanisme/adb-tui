package adb

import (
	"testing"
)

func TestKeycodeConstants(t *testing.T) {
	cases := []struct {
		name  string
		value int
	}{
		{"KeyHome", KeyHome},
		{"KeyBack", KeyBack},
		{"KeyCall", KeyCall},
		{"KeyEndCall", KeyEndCall},
		{"KeyDPadUp", KeyDPadUp},
		{"KeyDPadDown", KeyDPadDown},
		{"KeyDPadLeft", KeyDPadLeft},
		{"KeyDPadRight", KeyDPadRight},
		{"KeyDPadCenter", KeyDPadCenter},
		{"KeyVolumeUp", KeyVolumeUp},
		{"KeyVolumeDown", KeyVolumeDown},
		{"KeyPower", KeyPower},
		{"KeyCamera", KeyCamera},
		{"KeyClear", KeyClear},
		{"KeyMenu", KeyMenu},
		{"KeySearch", KeySearch},
		{"KeyMediaPlay", KeyMediaPlay},
		{"KeyMediaStop", KeyMediaStop},
		{"KeyMediaNext", KeyMediaNext},
		{"KeyMediaPrev", KeyMediaPrev},
		{"KeyMute", KeyMute},
		{"KeyTab", KeyTab},
		{"KeyEnter", KeyEnter},
		{"KeyDelete", KeyDelete},
		{"KeyRecents", KeyRecents},
		{"KeyBrightDown", KeyBrightDown},
		{"KeyBrightUp", KeyBrightUp},
		{"KeySleep", KeySleep},
		{"KeyWakeUp", KeyWakeUp},
	}

	for _, tc := range cases {
		if tc.value <= 0 {
			t.Errorf("%s should be positive, got %d", tc.name, tc.value)
		}
	}
}

func TestKeycodeSpecificValues(t *testing.T) {
	if KeyHome != 3 {
		t.Fatalf("expected KeyHome=3, got %d", KeyHome)
	}
	if KeyBack != 4 {
		t.Fatalf("expected KeyBack=4, got %d", KeyBack)
	}
	if KeyPower != 26 {
		t.Fatalf("expected KeyPower=26, got %d", KeyPower)
	}
	if KeyEnter != 66 {
		t.Fatalf("expected KeyEnter=66, got %d", KeyEnter)
	}
	if KeyDelete != 67 {
		t.Fatalf("expected KeyDelete=67, got %d", KeyDelete)
	}
	if KeyRecents != 187 {
		t.Fatalf("expected KeyRecents=187, got %d", KeyRecents)
	}
}

func TestKeycodeUniqueness(t *testing.T) {
	codes := map[int]string{}
	all := []struct {
		name  string
		value int
	}{
		{"KeyHome", KeyHome},
		{"KeyBack", KeyBack},
		{"KeyCall", KeyCall},
		{"KeyEndCall", KeyEndCall},
		{"KeyDPadUp", KeyDPadUp},
		{"KeyDPadDown", KeyDPadDown},
		{"KeyDPadLeft", KeyDPadLeft},
		{"KeyDPadRight", KeyDPadRight},
		{"KeyDPadCenter", KeyDPadCenter},
		{"KeyVolumeUp", KeyVolumeUp},
		{"KeyVolumeDown", KeyVolumeDown},
		{"KeyPower", KeyPower},
		{"KeyCamera", KeyCamera},
		{"KeyClear", KeyClear},
		{"KeyMenu", KeyMenu},
		{"KeySearch", KeySearch},
		{"KeyMediaPlay", KeyMediaPlay},
		{"KeyMediaStop", KeyMediaStop},
		{"KeyMediaNext", KeyMediaNext},
		{"KeyMediaPrev", KeyMediaPrev},
		{"KeyMute", KeyMute},
		{"KeyTab", KeyTab},
		{"KeyEnter", KeyEnter},
		{"KeyDelete", KeyDelete},
		{"KeyRecents", KeyRecents},
		{"KeyBrightDown", KeyBrightDown},
		{"KeyBrightUp", KeyBrightUp},
		{"KeySleep", KeySleep},
		{"KeyWakeUp", KeyWakeUp},
	}

	for _, k := range all {
		if prev, exists := codes[k.value]; exists {
			t.Fatalf("duplicate keycode %d: %s and %s", k.value, prev, k.name)
		}
		codes[k.value] = k.name
	}
}
