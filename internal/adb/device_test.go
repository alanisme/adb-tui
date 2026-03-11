package adb

import (
	"testing"
)

func TestParseDeviceLine_Full(t *testing.T) {
	line := "emulator-5554          device product:sdk_gphone64_arm64 model:sdk_gphone64_arm64 device:emu64a transport_id:1"
	dev := parseDeviceLine(line)
	if dev == nil {
		t.Fatal("expected device, got nil")
	}
	if dev.Serial != "emulator-5554" {
		t.Fatalf("expected emulator-5554, got %s", dev.Serial)
	}
	if dev.State != StateDevice {
		t.Fatalf("expected device state, got %s", dev.State)
	}
	if dev.Product != "sdk_gphone64_arm64" {
		t.Fatalf("expected sdk_gphone64_arm64, got %s", dev.Product)
	}
	if dev.Model != "sdk_gphone64_arm64" {
		t.Fatalf("expected sdk_gphone64_arm64, got %s", dev.Model)
	}
	if dev.Device != "emu64a" {
		t.Fatalf("expected emu64a, got %s", dev.Device)
	}
	if dev.TransportID != "1" {
		t.Fatalf("expected transport_id 1, got %s", dev.TransportID)
	}
}

func TestParseDeviceLine_Unauthorized(t *testing.T) {
	line := "HVA0T18C14000528 unauthorized usb:1-1 transport_id:3"
	dev := parseDeviceLine(line)
	if dev == nil {
		t.Fatal("expected device, got nil")
	}
	if dev.State != StateUnauthorized {
		t.Fatalf("expected unauthorized, got %s", dev.State)
	}
	if dev.TransportID != "3" {
		t.Fatalf("expected transport_id 3, got %s", dev.TransportID)
	}
}

func TestParseDeviceLine_Offline(t *testing.T) {
	line := "192.168.1.100:5555 offline"
	dev := parseDeviceLine(line)
	if dev == nil {
		t.Fatal("expected device, got nil")
	}
	if dev.Serial != "192.168.1.100:5555" {
		t.Fatalf("expected 192.168.1.100:5555, got %s", dev.Serial)
	}
	if dev.State != StateOffline {
		t.Fatalf("expected offline, got %s", dev.State)
	}
}

func TestParseDeviceLine_TooFewFields(t *testing.T) {
	dev := parseDeviceLine("single")
	if dev != nil {
		t.Fatal("expected nil for single field")
	}
}

func TestParseDeviceLine_Empty(t *testing.T) {
	dev := parseDeviceLine("")
	if dev != nil {
		t.Fatal("expected nil for empty line")
	}
}

func TestParseDeviceLine_NoExtraFields(t *testing.T) {
	line := "ABC123 device"
	dev := parseDeviceLine(line)
	if dev == nil {
		t.Fatal("expected device")
	}
	if dev.Serial != "ABC123" {
		t.Fatalf("expected ABC123, got %s", dev.Serial)
	}
	if dev.Model != "" {
		t.Fatalf("expected empty model, got %s", dev.Model)
	}
}

func TestParseDeviceLine_UnknownFields(t *testing.T) {
	line := "SER123 device foo:bar baz"
	dev := parseDeviceLine(line)
	if dev == nil {
		t.Fatal("expected device")
	}
	if dev.Serial != "SER123" {
		t.Fatalf("expected SER123, got %s", dev.Serial)
	}
}

func TestDeviceStates(t *testing.T) {
	if StateDevice != "device" {
		t.Fatal("unexpected StateDevice")
	}
	if StateOffline != "offline" {
		t.Fatal("unexpected StateOffline")
	}
	if StateUnauthorized != "unauthorized" {
		t.Fatal("unexpected StateUnauthorized")
	}
	if StateNoDevice != "no device" {
		t.Fatal("unexpected StateNoDevice")
	}
}

func TestParseDeviceLine_MultipleDeviceFormats(t *testing.T) {
	cases := []struct {
		name   string
		line   string
		serial string
		state  DeviceState
	}{
		{
			name:   "usb device",
			line:   "R5CR10FHKWN device usb:1-1 product:a52qnsxx model:SM_A528B device:a52q transport_id:2",
			serial: "R5CR10FHKWN",
			state:  StateDevice,
		},
		{
			name:   "wifi device",
			line:   "192.168.0.10:5555 device product:oriole model:Pixel_6 device:oriole transport_id:5",
			serial: "192.168.0.10:5555",
			state:  StateDevice,
		},
		{
			name:   "emulator",
			line:   "emulator-5556 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64x transport_id:4",
			serial: "emulator-5556",
			state:  StateDevice,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dev := parseDeviceLine(tc.line)
			if dev == nil {
				t.Fatal("expected device")
			}
			if dev.Serial != tc.serial {
				t.Fatalf("expected serial %s, got %s", tc.serial, dev.Serial)
			}
			if dev.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, dev.State)
			}
		})
	}
}

func TestNewClientWithPath(t *testing.T) {
	c := NewClientWithPath("/usr/bin/adb")
	if c.AdbPath() != "/usr/bin/adb" {
		t.Fatalf("expected /usr/bin/adb, got %s", c.AdbPath())
	}
}
