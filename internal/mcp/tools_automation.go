package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alanisme/adb-tui/internal/adb"
)

func RegisterAutomationTools(s *Server, client *adb.Client) {
	registerUITools(s, client)
	registerDeviceControlTools(s, client)
	registerDisplayTools(s, client)
	registerNotificationTools(s, client)
	registerActivityTools(s, client)
	registerFileIOTools(s, client)
	registerBatterySimTools(s, client)
}

func registerUITools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "get_ui_hierarchy",
			Description: "Dump the current UI hierarchy as XML from uiautomator. Returns the full UI tree with element bounds, text, resource IDs, and properties.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			xmlData, err := client.DumpUIHierarchy(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(xmlData), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "find_element",
			Description: "Find UI elements matching text, resource ID, or class name in the current screen. Returns matching elements with their bounds and properties.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"text":{"type":"string","description":"Text or content description to search for (case-insensitive substring match)."},"resource_id":{"type":"string","description":"Resource ID to match (exact match)."},"class_name":{"type":"string","description":"Class name suffix to match (e.g. Button, TextView)."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				Text       string
				ResourceID string `json:"resource_id"`
				ClassName  string `json:"class_name"`
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			xmlData, err := client.DumpUIHierarchy(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			h, err := adb.ParseUIHierarchy(xmlData)
			if err != nil {
				return nil, err
			}
			elements := adb.FlattenNodes(h.Nodes)
			matches := adb.FindElements(elements, p.Text, p.ResourceID, p.ClassName)
			if len(matches) == 0 {
				return textResult("No matching elements found."), nil
			}
			// Wrap with index for tap_element reference
			type indexed struct {
				Index   int            `json:"index"`
				Element adb.UIElement  `json:"element"`
				CenterX int           `json:"center_x"`
				CenterY int           `json:"center_y"`
			}
			results := make([]indexed, len(matches))
			for i, m := range matches {
				cx, cy := m.Bounds.Center()
				results[i] = indexed{Index: i, Element: m, CenterX: cx, CenterY: cy}
			}
			data, err := json.Marshal(results)
			if err != nil {
				return nil, err
			}
			return textResult(string(data)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "tap_element",
			Description: "Find a UI element by text or resource ID and tap its center. Combines UI dump, element search, and tap into one action. Use index to select among multiple matches (sorted top-to-bottom, left-to-right). Use find_element first to discover available elements and their indices.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"text":{"type":"string","description":"Text or content description to search for (case-insensitive substring)."},"resource_id":{"type":"string","description":"Resource ID to match (exact)."},"index":{"type":"integer","description":"0-based index when multiple elements match. Default 0 (first match). Elements are sorted by position: top-to-bottom, left-to-right."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				Text       string
				ResourceID string `json:"resource_id"`
				Index      int    `json:"index"`
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.FindAndTapElement(ctx, p.Serial, p.Text, p.ResourceID, p.Index); err != nil {
				return nil, err
			}
			return textResult("Element tapped."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "wait_for_element",
			Description: "Wait for a UI element matching text or resource ID to appear on screen. Polls the UI hierarchy at 500ms intervals until the element is found or timeout.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"text":{"type":"string","description":"Text to wait for."},"resource_id":{"type":"string","description":"Resource ID to wait for."},"timeout_ms":{"type":"integer","description":"Maximum wait time in milliseconds. Defaults to 10000."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				Text       string
				ResourceID string `json:"resource_id"`
				TimeoutMS  int    `json:"timeout_ms"`
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			timeout := p.TimeoutMS
			if timeout <= 0 {
				timeout = 10000
			}
			elem, err := client.WaitForElement(ctx, p.Serial, p.Text, p.ResourceID, timeout)
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(elem)
			if err != nil {
				return nil, err
			}
			return textResult(string(data)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_focused_app",
			Description: "Get the package name of the currently focused application.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			pkg, err := client.GetFocusedApp(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(pkg), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_current_activity",
			Description: "Get the currently resumed activity (package/activity component).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			activity, err := client.GetCurrentActivity(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(activity), nil
		},
	)
}

func registerDeviceControlTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "screen_on",
			Description: "Wake the device screen.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ScreenOn(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Screen turned on."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "screen_off",
			Description: "Turn off the device screen.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ScreenOff(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Screen turned off."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "is_screen_on",
			Description: "Check whether the device screen is currently on.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			on, err := client.IsScreenOn(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if on {
				return textResult("Screen is ON."), nil
			}
			return textResult("Screen is OFF."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_brightness",
			Description: "Get the current screen brightness level (0-255).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			level, err := client.GetBrightness(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("%d", level)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_brightness",
			Description: "Set the screen brightness level (0-255). Disables auto-brightness.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"level":{"type":"integer","description":"Brightness level (0-255)."}},"required":["level"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Level  int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetBrightness(ctx, p.Serial, p.Level); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Brightness set to %d.", p.Level)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_rotation",
			Description: "Set screen rotation. Disables auto-rotation. Values: 0=natural, 1=90 degrees, 2=180 degrees, 3=270 degrees.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"rotation":{"type":"integer","description":"Rotation value: 0=natural, 1=90°, 2=180°, 3=270°.","enum":[0,1,2,3]}},"required":["rotation"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial   string
				Rotation int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetRotation(ctx, p.Serial, p.Rotation); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Rotation set to %d.", p.Rotation)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_auto_rotation",
			Description: "Enable or disable auto-rotation.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"enabled":{"type":"boolean","description":"True to enable auto-rotation, false to disable."}},"required":["enabled"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Enabled bool
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetAutoRotation(ctx, p.Serial, p.Enabled); err != nil {
				return nil, err
			}
			if p.Enabled {
				return textResult("Auto-rotation enabled."), nil
			}
			return textResult("Auto-rotation disabled."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_airplane_mode",
			Description: "Check whether airplane mode is enabled.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			on, err := client.GetAirplaneMode(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if on {
				return textResult("Airplane mode is ON."), nil
			}
			return textResult("Airplane mode is OFF."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_airplane_mode",
			Description: "Enable or disable airplane mode.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"enabled":{"type":"boolean","description":"True to enable airplane mode, false to disable."}},"required":["enabled"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Enabled bool
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetAirplaneMode(ctx, p.Serial, p.Enabled); err != nil {
				return nil, err
			}
			if p.Enabled {
				return textResult("Airplane mode enabled."), nil
			}
			return textResult("Airplane mode disabled."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "volume_up",
			Description: "Increase media volume by one step.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.VolumeUp(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Volume increased."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "volume_down",
			Description: "Decrease media volume by one step.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.VolumeDown(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Volume decreased."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "volume_mute",
			Description: "Toggle mute state.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.VolumeMute(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Mute toggled."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "media_play",
			Description: "Send media play key event.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.MediaPlay(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Media play sent."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "media_pause",
			Description: "Send media pause key event.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.MediaPause(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Media pause sent."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "media_next",
			Description: "Send media next track key event.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.MediaNext(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Media next sent."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "media_previous",
			Description: "Send media previous track key event.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.MediaPrev(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Media previous sent."), nil
		},
	)
}

func registerDisplayTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "set_display_size",
			Description: "Override the display resolution. Use reset_display_size to restore default.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"width":{"type":"integer","description":"Width in pixels."},"height":{"type":"integer","description":"Height in pixels."}},"required":["width","height"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Width  int
				Height int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetDisplaySize(ctx, p.Serial, p.Width, p.Height); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Display size set to %dx%d.", p.Width, p.Height)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reset_display_size",
			Description: "Reset display resolution to the physical default.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ResetDisplaySize(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Display size reset."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_density",
			Description: "Override the display density (DPI). Use reset_density to restore default.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"dpi":{"type":"integer","description":"Density in DPI."}},"required":["dpi"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				DPI    int `json:"dpi"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetDensity(ctx, p.Serial, p.DPI); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Density set to %d DPI.", p.DPI)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reset_density",
			Description: "Reset display density to the physical default.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ResetDensity(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Density reset."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_font_scale",
			Description: "Get the current font scale factor.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			scale, err := client.GetFontScale(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("%.2f", scale)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_font_scale",
			Description: "Set the font scale factor (e.g. 1.0 for default, 1.5 for large).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"scale":{"type":"number","description":"Font scale factor (e.g. 0.85, 1.0, 1.15, 1.3)."}},"required":["scale"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Scale  float64
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetFontScale(ctx, p.Serial, p.Scale); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Font scale set to %.2f.", p.Scale)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_screen_size",
			Description: "Get the physical screen resolution in pixels.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			w, h, err := client.GetScreenSize(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("%dx%d", w, h)), nil
		},
	)
}

func registerNotificationTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "list_notifications",
			Description: "List active notifications on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			notifs, err := client.ListNotifications(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if len(notifs) == 0 {
				return textResult("No notifications."), nil
			}
			var sb strings.Builder
			for _, n := range notifs {
				fmt.Fprintf(&sb, "[%s] %s: %s\n", n.Package, n.Title, n.Text)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "expand_notifications",
			Description: "Pull down the notification shade.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ExpandNotifications(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Notification shade expanded."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "collapse_notifications",
			Description: "Close the notification shade.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.CollapseNotifications(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Notification shade collapsed."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "expand_quick_settings",
			Description: "Pull down the quick settings panel.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ExpandQuickSettings(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Quick settings expanded."), nil
		},
	)
}

func registerActivityTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "list_activities",
			Description: "List activities declared by a package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"package":{"type":"string","description":"Package name."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			activities, err := client.ListActivities(ctx, p.Serial, p.Package)
			if err != nil {
				return nil, err
			}
			if len(activities) == 0 {
				return textResult("No activities found."), nil
			}
			return textResult(strings.Join(activities, "\n")), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "open_url",
			Description: "Open a URL in the device's default browser.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"url":{"type":"string","description":"URL to open."}},"required":["url"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				URL    string `json:"url"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.OpenURL(ctx, p.Serial, p.URL); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Opened %s.", p.URL)), nil
		},
	)
}

func registerFileIOTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "read_file",
			Description: "Read the content of a text file on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."}},"required":["path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			content, err := client.Cat(ctx, p.Serial, p.Path)
			if err != nil {
				return nil, err
			}
			return textResult(content), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "write_file",
			Description: "Write text content to a file on the device. Uses base64 encoding for safe transfer.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."},"content":{"type":"string","description":"Text content to write."}},"required":["path","content"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Path    string
				Content string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.WriteFile(ctx, p.Serial, p.Path, p.Content); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Written to %s.", p.Path)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "stat_file",
			Description: "Get file status information (permissions, size, modification time).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."}},"required":["path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			info, err := client.Stat(ctx, p.Serial, p.Path)
			if err != nil {
				return nil, err
			}
			typeStr := "file"
			if info.IsDir {
				typeStr = "directory"
			} else if info.IsLink {
				typeStr = "symlink"
			}
			return textResult(fmt.Sprintf("Type: %s\nPermissions: %s\nSize: %d\nModified: %s",
				typeStr, info.Permissions, info.Size, info.ModTime.Format("2006-01-02 15:04:05"))), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "delete_file",
			Description: "Delete a file or directory on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."}},"required":["path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Remove(ctx, p.Serial, p.Path); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Deleted %s.", p.Path)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "mkdir",
			Description: "Create a directory on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"Directory path to create."}},"required":["path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Mkdir(ctx, p.Serial, p.Path); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Created directory %s.", p.Path)), nil
		},
	)
}

func registerBatterySimTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "simulate_battery",
			Description: "Simulate battery conditions for testing. Set level, status, or plugged state. Use reset_battery to restore real values.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"level":{"type":"integer","description":"Battery level (0-100)."},"status":{"type":"integer","description":"Battery status: 1=unknown, 2=charging, 3=discharging, 4=not charging, 5=full."},"plugged":{"type":"integer","description":"Plug type: 0=none, 1=AC, 2=USB, 4=wireless."},"unplug":{"type":"boolean","description":"Simulate unplugged state."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Level   *int `json:"level"`
				Status  *int `json:"status"`
				Plugged *int `json:"plugged"`
				Unplug  bool `json:"unplug"`
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			var actions []string
			if p.Unplug {
				if err := client.SimulateBatteryUnplug(ctx, p.Serial); err != nil {
					return nil, err
				}
				actions = append(actions, "unplugged")
			}
			if p.Level != nil {
				if err := client.SetBatteryLevel(ctx, p.Serial, *p.Level); err != nil {
					return nil, err
				}
				actions = append(actions, fmt.Sprintf("level=%d", *p.Level))
			}
			if p.Status != nil {
				if err := client.SetBatteryStatus(ctx, p.Serial, *p.Status); err != nil {
					return nil, err
				}
				actions = append(actions, fmt.Sprintf("status=%d", *p.Status))
			}
			if p.Plugged != nil {
				if err := client.SetBatteryPlugged(ctx, p.Serial, *p.Plugged); err != nil {
					return nil, err
				}
				actions = append(actions, fmt.Sprintf("plugged=%d", *p.Plugged))
			}
			if len(actions) == 0 {
				return textResult("No battery changes specified."), nil
			}
			return textResult(fmt.Sprintf("Battery simulation: %s.", strings.Join(actions, ", "))), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reset_battery",
			Description: "Reset battery simulation and restore real battery values.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ResetBattery(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Battery simulation reset."), nil
		},
	)
}
