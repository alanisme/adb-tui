package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alanisme/adb-tui/internal/adb"
)

func newTestServer() *Server {
	s := NewServer("test", "1.0")
	client := adb.NewClientWithPath("/dev/null")
	RegisterADBTools(s, client)
	return s
}

func TestAllToolsRegistered(t *testing.T) {
	s := newTestServer()

	expected := []string{
		// Device
		"list_devices", "device_info", "connect", "disconnect",
		"get_device_state", "is_rooted", "remount",
		// Shell
		"shell",
		// Packages
		"list_packages", "install_apk", "uninstall_package", "package_info",
		"force_stop", "clear_data", "enable_package", "disable_package",
		"grant_permission", "revoke_permission",
		// Files
		"push_file", "pull_file", "list_files", "find_files",
		"disk_usage", "chmod", "chown",
		// Screen
		"screenshot", "screen_record",
		// Logcat
		"logcat", "clear_logcat",
		// Input
		"tap", "swipe", "key_event", "input_text",
		"long_press", "human_swipe",
		// Properties
		"get_prop", "set_prop", "list_props",
		// Port Forward
		"forward", "forward_list", "forward_remove", "forward_remove_all",
		"reverse", "reverse_list", "reverse_remove_all",
		// Intent
		"start_activity", "send_broadcast",
		// System
		"reboot", "get_battery",
		// Connectivity
		"wifi_control", "get_ip_address", "get_network_info",
		"tcpip_mode", "usb_mode",
		// Settings
		"get_setting", "put_setting", "list_settings", "delete_setting",
		// Process
		"list_processes", "kill_process", "kill_process_by_name",
		"memory_info", "app_memory_info", "top_processes",
		// Dumpsys
		"dumpsys", "dumpsys_list", "battery_info", "display_info", "window_info",
		// Security
		"selinux_status", "set_selinux", "list_permissions", "get_apk_path",
		// Testing
		"run_monkey", "run_instrumentation",
		// Backup
		"bugreport", "sideload", "backup", "restore",
		// Network
		"netstat", "ping_host",
		// Clipboard
		"get_clipboard", "set_clipboard",
		// Extended shell
		"root", "unroot",
		// UI Automation
		"get_ui_hierarchy", "find_element", "tap_element",
		"wait_for_element", "get_focused_app", "get_current_activity",
		// Device Control
		"screen_on", "screen_off", "is_screen_on",
		"get_brightness", "set_brightness",
		"set_rotation", "set_auto_rotation",
		"get_airplane_mode", "set_airplane_mode",
		"volume_up", "volume_down", "volume_mute",
		"media_play", "media_pause", "media_next", "media_previous",
		// Display
		"set_display_size", "reset_display_size",
		"set_density", "reset_density",
		"get_font_scale", "set_font_scale", "get_screen_size",
		// Notifications
		"list_notifications", "expand_notifications",
		"collapse_notifications", "expand_quick_settings",
		// Activity
		"list_activities", "open_url",
		// File I/O
		"read_file", "write_file", "stat_file", "delete_file", "mkdir",
		// Battery Simulation
		"simulate_battery", "reset_battery",
	}

	for _, name := range expected {
		if _, ok := s.tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
}

func TestToolCount(t *testing.T) {
	s := newTestServer()
	count := len(s.tools)
	if count < 115 {
		t.Errorf("expected at least 115 tools, got %d", count)
	}
}

func TestAllToolsHaveValidSchema(t *testing.T) {
	s := newTestServer()
	for name, entry := range s.tools {
		if entry.tool.InputSchema == nil {
			t.Errorf("tool %q has nil InputSchema", name)
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(entry.tool.InputSchema, &schema); err != nil {
			t.Errorf("tool %q has invalid InputSchema: %v", name, err)
		}
	}
}

func TestAllToolsHaveDescription(t *testing.T) {
	s := newTestServer()
	for name, entry := range s.tools {
		if entry.tool.Description == "" {
			t.Errorf("tool %q has empty description", name)
		}
	}
}

func TestAllToolsHaveHandler(t *testing.T) {
	s := newTestServer()
	for name, entry := range s.tools {
		if entry.handler == nil {
			t.Errorf("tool %q has nil handler", name)
		}
	}
}

func TestToolCallWithInvalidJSON(t *testing.T) {
	s := newTestServer()
	toolsNeedingParams := []string{
		"connect", "install_apk", "uninstall_package", "force_stop",
		"push_file", "pull_file", "screenshot", "tap", "swipe",
		"get_prop", "set_prop", "forward", "forward_remove",
		"set_selinux", "get_apk_path", "set_clipboard", "long_press",
		// Automation tools with required params
		"set_brightness", "set_rotation", "set_auto_rotation",
		"set_airplane_mode", "set_display_size", "set_density",
		"set_font_scale", "list_activities", "open_url",
		"read_file", "write_file", "stat_file", "delete_file", "mkdir",
	}

	for _, name := range toolsNeedingParams {
		entry, ok := s.tools[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		_, err := entry.handler(context.Background(), json.RawMessage(`{invalid`))
		if err == nil {
			t.Errorf("tool %q should return error for invalid JSON", name)
		}
	}
}

func TestToolCallWithNilParams(t *testing.T) {
	s := newTestServer()
	toolsAcceptingNil := []string{
		"list_devices", "clear_logcat", "list_props",
		"forward_list", "forward_remove_all",
		"reverse_list", "reverse_remove_all",
		"get_device_state", "is_rooted", "remount",
		"get_clipboard", "get_ip_address", "get_network_info",
		"usb_mode", "tcpip_mode",
		"selinux_status", "memory_info",
		"list_processes", "top_processes",
		"dumpsys_list",
		// Automation tools accepting nil
		"get_ui_hierarchy", "find_element", "tap_element",
		"wait_for_element", "get_focused_app", "get_current_activity",
		"screen_on", "screen_off", "is_screen_on",
		"get_brightness", "get_airplane_mode",
		"volume_up", "volume_down", "volume_mute",
		"media_play", "media_pause", "media_next", "media_previous",
		"reset_display_size", "reset_density",
		"get_font_scale", "get_screen_size",
		"list_notifications", "expand_notifications",
		"collapse_notifications", "expand_quick_settings",
		"simulate_battery", "reset_battery",
	}

	for _, name := range toolsAcceptingNil {
		entry, ok := s.tools[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		// These will fail because adb binary is /dev/null, but they should
		// not panic. We just verify the handler doesn't crash on nil params.
		entry.handler(context.Background(), nil)
	}
}

func TestToolsListViaServer(t *testing.T) {
	s := newTestServer()
	req := makeRequest(MethodToolsList, newTestID(1), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var result ListToolsResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Tools) < 115 {
		t.Fatalf("expected at least 115 tools in list, got %d", len(result.Tools))
	}
}

func TestNewToolsRegistered(t *testing.T) {
	s := newTestServer()

	newTools := map[string]string{
		"get_device_state":   "Get the current state",
		"is_rooted":          "Check if the device has root",
		"remount":            "Remount device filesystem",
		"get_clipboard":      "Get the current clipboard",
		"set_clipboard":      "Set the clipboard",
		"long_press":         "Perform a long press",
		"human_swipe":        "Perform a natural swipe",
		"forward_remove_all": "Remove all port forwards",
		"reverse_list":       "List all active reverse",
		"reverse_remove_all": "Remove all reverse",
		"tcpip_mode":         "Switch device to TCP/IP",
		"usb_mode":           "Switch device back to USB",
		"get_ip_address":     "Get the IP address",
		"get_network_info":   "Get network interface",
	}

	for name, descPrefix := range newTools {
		entry, ok := s.tools[name]
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if entry.tool.Description == "" {
			t.Errorf("tool %q has empty description", name)
		}
		_ = descPrefix
	}
}

func TestLongPressDefaultDuration(t *testing.T) {
	s := newTestServer()
	entry := s.tools["long_press"]

	// Verify schema has duration_ms field
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	if _, ok := props["duration_ms"]; !ok {
		t.Error("long_press schema missing duration_ms property")
	}
	if _, ok := props["x"]; !ok {
		t.Error("long_press schema missing x property")
	}
	if _, ok := props["y"]; !ok {
		t.Error("long_press schema missing y property")
	}
}

func TestHumanSwipeSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["human_swipe"]

	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)

	requiredFields := []string{
		"screen_width", "screen_height",
		"start_x", "start_y", "end_x", "end_y",
	}
	for _, field := range requiredFields {
		if _, ok := props[field]; !ok {
			t.Errorf("human_swipe schema missing %q property", field)
		}
	}

	required := schema["required"].([]any)
	requiredSet := make(map[string]bool)
	for _, r := range required {
		requiredSet[r.(string)] = true
	}
	for _, field := range requiredFields {
		if !requiredSet[field] {
			t.Errorf("human_swipe schema should require %q", field)
		}
	}
}

func TestTcpIpModeSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["tcpip_mode"]

	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	if _, ok := props["port"]; !ok {
		t.Error("tcpip_mode schema missing port property")
	}
}

func TestClipboardToolsSchema(t *testing.T) {
	s := newTestServer()

	// get_clipboard requires no params
	getEntry := s.tools["get_clipboard"]
	var getSchema map[string]any
	json.Unmarshal(getEntry.tool.InputSchema, &getSchema)
	if _, ok := getSchema["required"]; ok {
		t.Error("get_clipboard should not have required params")
	}

	// set_clipboard requires text
	setEntry := s.tools["set_clipboard"]
	var setSchema map[string]any
	json.Unmarshal(setEntry.tool.InputSchema, &setSchema)
	required, ok := setSchema["required"].([]any)
	if !ok || len(required) == 0 {
		t.Error("set_clipboard should require text param")
	}
}

func TestForwardToolsCompleteness(t *testing.T) {
	s := newTestServer()
	forwardTools := []string{
		"forward", "forward_list", "forward_remove", "forward_remove_all",
		"reverse", "reverse_list", "reverse_remove_all",
	}
	for _, name := range forwardTools {
		if _, ok := s.tools[name]; !ok {
			t.Errorf("expected port forward tool %q to be registered", name)
		}
	}
}

func TestAutomationToolsRegistered(t *testing.T) {
	s := newTestServer()

	categories := map[string][]string{
		"UI": {
			"get_ui_hierarchy", "find_element", "tap_element",
			"wait_for_element", "get_focused_app", "get_current_activity",
		},
		"Device Control": {
			"screen_on", "screen_off", "is_screen_on",
			"get_brightness", "set_brightness",
			"set_rotation", "set_auto_rotation",
			"get_airplane_mode", "set_airplane_mode",
			"volume_up", "volume_down", "volume_mute",
			"media_play", "media_pause", "media_next", "media_previous",
		},
		"Display": {
			"set_display_size", "reset_display_size",
			"set_density", "reset_density",
			"get_font_scale", "set_font_scale", "get_screen_size",
		},
		"Notifications": {
			"list_notifications", "expand_notifications",
			"collapse_notifications", "expand_quick_settings",
		},
		"Activity": {
			"list_activities", "open_url",
		},
		"File I/O": {
			"read_file", "write_file", "stat_file", "delete_file", "mkdir",
		},
		"Battery Sim": {
			"simulate_battery", "reset_battery",
		},
	}

	for category, tools := range categories {
		for _, name := range tools {
			entry, ok := s.tools[name]
			if !ok {
				t.Errorf("[%s] tool %q not registered", category, name)
				continue
			}
			if entry.tool.Description == "" {
				t.Errorf("[%s] tool %q has empty description", category, name)
			}
			if entry.handler == nil {
				t.Errorf("[%s] tool %q has nil handler", category, name)
			}
		}
	}
}

func TestBrightnessSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["set_brightness"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	if _, ok := props["level"]; !ok {
		t.Error("set_brightness schema missing level property")
	}
	required := schema["required"].([]any)
	if len(required) == 0 {
		t.Error("set_brightness should have required params")
	}
}

func TestRotationSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["set_rotation"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	rotProp := props["rotation"].(map[string]any)
	if rotProp["type"] != "integer" {
		t.Error("rotation should be integer type")
	}
	enumVals, ok := rotProp["enum"].([]any)
	if !ok || len(enumVals) != 4 {
		t.Error("rotation should have 4 enum values")
	}
}

func TestDisplaySizeSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["set_display_size"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	if _, ok := props["width"]; !ok {
		t.Error("set_display_size schema missing width")
	}
	if _, ok := props["height"]; !ok {
		t.Error("set_display_size schema missing height")
	}
}

func TestSimulateBatterySchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["simulate_battery"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	props := schema["properties"].(map[string]any)
	for _, field := range []string{"level", "status", "plugged", "unplug"} {
		if _, ok := props[field]; !ok {
			t.Errorf("simulate_battery schema missing %q", field)
		}
	}
}

func TestWriteFileSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["write_file"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	required := schema["required"].([]any)
	requiredSet := make(map[string]bool)
	for _, r := range required {
		requiredSet[r.(string)] = true
	}
	if !requiredSet["path"] || !requiredSet["content"] {
		t.Error("write_file should require path and content")
	}
}

func TestOpenURLSchema(t *testing.T) {
	s := newTestServer()
	entry := s.tools["open_url"]
	var schema map[string]any
	json.Unmarshal(entry.tool.InputSchema, &schema)
	required := schema["required"].([]any)
	if len(required) == 0 {
		t.Error("open_url should require url param")
	}
}
