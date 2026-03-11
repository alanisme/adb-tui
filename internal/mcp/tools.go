package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alanisme/adb-tui/internal/adb"
)

func RegisterADBTools(s *Server, client *adb.Client) {
	registerDeviceTools(s, client)
	registerShellTools(s, client)
	registerPackageTools(s, client)
	registerFileTools(s, client)
	registerScreenTools(s, client)
	registerLogcatTools(s, client)
	registerInputTools(s, client)
	registerPropertyTools(s, client)
	registerForwardTools(s, client)
	registerIntentTools(s, client)
	registerSystemTools(s, client)
	registerConnectivityTools(s, client)
	RegisterExtendedTools(s, client)
	RegisterAutomationTools(s, client)
}

func schema(s string) json.RawMessage { return json.RawMessage(s) }

func textResult(text string) *ToolCallResult {
	return &ToolCallResult{Content: []Content{TextContent(text)}}
}

func registerDeviceTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "list_devices",
			Description: "List connected Android devices with their serial numbers, state, and model information.",
			InputSchema: schema(`{"type":"object","properties":{}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			devices, err := client.ListDevices(ctx)
			if err != nil {
				return nil, err
			}
			if len(devices) == 0 {
				return textResult("No devices connected."), nil
			}
			var sb strings.Builder
			for _, d := range devices {
				fmt.Fprintf(&sb, "%s\t%s\tmodel:%s\tproduct:%s\tdevice:%s\n",
					d.Serial, d.State, d.Model, d.Product, d.Device)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "device_info",
			Description: "Get detailed information about a connected Android device including model, Android version, SDK level, and hardware details.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number. If omitted, uses the only connected device."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			info, err := client.GetDeviceInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "Model: %s\n", info.Model)
			fmt.Fprintf(&sb, "Brand: %s\n", info.Brand)
			fmt.Fprintf(&sb, "Manufacturer: %s\n", info.Manufacturer)
			fmt.Fprintf(&sb, "Product: %s\n", info.Product)
			fmt.Fprintf(&sb, "Hardware: %s\n", info.Hardware)
			fmt.Fprintf(&sb, "Android Version: %s\n", info.AndroidVersion)
			fmt.Fprintf(&sb, "SDK Version: %s\n", info.SDKVersion)
			fmt.Fprintf(&sb, "Build Number: %s\n", info.BuildNumber)
			fmt.Fprintf(&sb, "Serial: %s\n", info.Serial)
			if len(info.ABIs) > 0 {
				fmt.Fprintf(&sb, "ABIs: %s\n", strings.Join(info.ABIs, ", "))
			}
			fmt.Fprintf(&sb, "Display: %s @ %s dpi\n", info.DisplaySize, info.DisplayDensity)
			fmt.Fprintf(&sb, "IP Address: %s\n", info.IPAddress)
			fmt.Fprintf(&sb, "MAC Address: %s\n", info.MacAddress)
			fmt.Fprintf(&sb, "Battery: %s%% (%s)\n", info.BatteryLevel, info.BatteryStatus)
			fmt.Fprintf(&sb, "RAM: %s total, %s available\n", info.TotalRAM, info.AvailableRAM)
			fmt.Fprintf(&sb, "Storage: %s total, %s available\n", info.TotalStorage, info.AvailableStorage)
			fmt.Fprintf(&sb, "Uptime: %s\n", info.Uptime)
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "connect",
			Description: "Connect to a device over TCP/IP using host:port.",
			InputSchema: schema(`{"type":"object","properties":{"host":{"type":"string","description":"Host and port to connect to (e.g. 192.168.1.100:5555)."}},"required":["host"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Host string }
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Connect(ctx, p.Host); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Connected to %s.", p.Host)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "disconnect",
			Description: "Disconnect from a device. If host is empty, disconnects all devices.",
			InputSchema: schema(`{"type":"object","properties":{"host":{"type":"string","description":"Host to disconnect. Leave empty to disconnect all."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Host string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.Disconnect(ctx, p.Host); err != nil {
				return nil, err
			}
			if p.Host == "" {
				return textResult("Disconnected all devices."), nil
			}
			return textResult(fmt.Sprintf("Disconnected %s.", p.Host)), nil
		},
	)
}

func registerShellTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "shell",
			Description: "Execute a shell command on an Android device and return the output.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"command":{"type":"string","description":"Shell command to execute."}},"required":["command"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Command string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			// NOTE: This tool intentionally passes the raw command to client.Shell()
			// as an escape hatch for arbitrary shell commands. Other tools use typed
			// ADB client methods that handle shell quoting internally.
			result, err := client.Shell(ctx, p.Serial, p.Command)
			if err != nil {
				return nil, err
			}
			return textResult(result.Output), nil
		},
	)
}

func registerPackageTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "install_apk",
			Description: "Install an APK file on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"path":{"type":"string","description":"Path to the APK file on the host."},"reinstall":{"type":"boolean","description":"Reinstall keeping data."},"downgrade":{"type":"boolean","description":"Allow version downgrade."}},"required":["path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Path      string
				Reinstall bool
				Downgrade bool
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			opts := adb.InstallOptions{
				Reinstall:      p.Reinstall,
				AllowDowngrade: p.Downgrade,
			}
			if err := client.InstallAPK(ctx, p.Serial, p.Path, opts); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Installed %s.", p.Path)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "uninstall_package",
			Description: "Uninstall a package from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name to uninstall."},"keep_data":{"type":"boolean","description":"Keep app data after uninstall."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial   string
				Package  string `json:"package"`
				KeepData bool   `json:"keep_data"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.UninstallPackage(ctx, p.Serial, p.Package, p.KeepData); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Uninstalled %s.", p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "list_packages",
			Description: "List installed packages on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"system":{"type":"boolean","description":"Show only system packages."},"third_party":{"type":"boolean","description":"Show only third-party packages."},"filter":{"type":"string","description":"Filter packages by keyword."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				System     bool   `json:"system"`
				ThirdParty bool   `json:"third_party"`
				Filter     string `json:"filter"`
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			opts := adb.ListOptions{
				ShowSystem:     p.System,
				ShowThirdParty: p.ThirdParty,
				Filter:         p.Filter,
			}
			packages, err := client.ListPackages(ctx, p.Serial, opts)
			if err != nil {
				return nil, err
			}
			if len(packages) == 0 {
				return textResult("No packages found."), nil
			}
			var sb strings.Builder
			for _, pkg := range packages {
				sb.WriteString(pkg.Name)
				sb.WriteByte('\n')
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "package_info",
			Description: "Get detailed information about an installed package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			detail, err := client.GetPackageInfo(ctx, p.Serial, p.Package)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "Package: %s\n", detail.Name)
			fmt.Fprintf(&sb, "Version: %s (%s)\n", detail.VersionName, detail.VersionCode)
			fmt.Fprintf(&sb, "Installer: %s\n", detail.Installer)
			fmt.Fprintf(&sb, "UID: %s\n", detail.UID)
			fmt.Fprintf(&sb, "Data Dir: %s\n", detail.DataDir)
			fmt.Fprintf(&sb, "APK Path: %s\n", detail.APKPath)
			fmt.Fprintf(&sb, "Enabled: %v\n", detail.Enabled)
			fmt.Fprintf(&sb, "System: %v\n", detail.System)
			fmt.Fprintf(&sb, "First Install: %s\n", detail.FirstInstall)
			fmt.Fprintf(&sb, "Last Update: %s\n", detail.LastUpdate)
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "force_stop",
			Description: "Force stop an application.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name to force stop."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.ForceStop(ctx, p.Serial, p.Package); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Force stopped %s.", p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "clear_data",
			Description: "Clear all data for an application.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name to clear data for."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.ClearData(ctx, p.Serial, p.Package); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Cleared data for %s.", p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "enable_package",
			Description: "Enable a disabled package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name to enable."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.EnablePackage(ctx, p.Serial, p.Package); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Enabled %s.", p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "disable_package",
			Description: "Disable a package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name to disable."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.DisablePackage(ctx, p.Serial, p.Package); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Disabled %s.", p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "grant_permission",
			Description: "Grant a runtime permission to a package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name."},"permission":{"type":"string","description":"Permission to grant (e.g. android.permission.CAMERA)."}},"required":["package","permission"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				Package    string `json:"package"`
				Permission string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.GrantPermission(ctx, p.Serial, p.Package, p.Permission); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Granted %s to %s.", p.Permission, p.Package)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "revoke_permission",
			Description: "Revoke a runtime permission from a package.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"package":{"type":"string","description":"Package name."},"permission":{"type":"string","description":"Permission to revoke."}},"required":["package","permission"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				Package    string `json:"package"`
				Permission string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.RevokePermission(ctx, p.Serial, p.Package, p.Permission); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Revoked %s from %s.", p.Permission, p.Package)), nil
		},
	)
}

func registerFileTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "push_file",
			Description: "Push a file from the host to the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"local":{"type":"string","description":"Local file path on the host."},"remote":{"type":"string","description":"Remote path on the device."}},"required":["local","remote"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Local  string
				Remote string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Push(ctx, p.Serial, p.Local, p.Remote); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Pushed %s to %s.", p.Local, p.Remote)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "pull_file",
			Description: "Pull a file from the device to the host.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"remote":{"type":"string","description":"Remote file path on the device."},"local":{"type":"string","description":"Local path on the host."}},"required":["remote","local"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Remote string
				Local  string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Pull(ctx, p.Serial, p.Remote, p.Local); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Pulled %s to %s.", p.Remote, p.Local)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "list_files",
			Description: "List files in a directory on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"path":{"type":"string","description":"Directory path on the device. Defaults to /sdcard."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if p.Path == "" {
				p.Path = "/sdcard"
			}
			files, err := client.ListDir(ctx, p.Serial, p.Path)
			if err != nil {
				return nil, err
			}
			if len(files) == 0 {
				return textResult("No files found."), nil
			}
			var sb strings.Builder
			for _, f := range files {
				typeChar := "-"
				if f.IsDir {
					typeChar = "d"
				} else if f.IsLink {
					typeChar = "l"
				}
				name := f.Name
				if f.IsLink && f.LinkTarget != "" {
					name = f.Name + " -> " + f.LinkTarget
				}
				fmt.Fprintf(&sb, "%s %s %8d %s %s\n",
					typeChar, f.Permissions, f.Size,
					f.ModTime.Format("2006-01-02 15:04"), name)
			}
			return textResult(sb.String()), nil
		},
	)
}

func registerScreenTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "screenshot",
			Description: "Take a screenshot and save it to the specified path on the host.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"output_path":{"type":"string","description":"Output file path on the host."}},"required":["output_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				OutputPath string `json:"output_path"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Screenshot(ctx, p.Serial, p.OutputPath); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Screenshot saved to %s.", p.OutputPath)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "screen_record",
			Description: "Record the device screen to a file.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"output_path":{"type":"string","description":"Output file path on the host."},"time_limit":{"type":"integer","description":"Recording time limit in seconds (max 180)."},"bit_rate":{"type":"integer","description":"Video bit rate in bits per second."}},"required":["output_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				OutputPath string `json:"output_path"`
				TimeLimit  int    `json:"time_limit"`
				BitRate    int    `json:"bit_rate"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			devicePath := "/sdcard/screenrecord_mcp.mp4"
			opts := adb.ScreenRecordOptions{
				TimeLimit: p.TimeLimit,
				BitRate:   p.BitRate,
			}
			cmd, err := client.ScreenRecord(ctx, p.Serial, devicePath, opts)
			if err != nil {
				return nil, fmt.Errorf("screenrecord failed: %w", err)
			}
			if err := cmd.Wait(); err != nil {
				return nil, fmt.Errorf("screenrecord failed: %w", err)
			}
			if err := client.Pull(ctx, p.Serial, devicePath, p.OutputPath); err != nil {
				return nil, fmt.Errorf("pull recording failed: %w", err)
			}
			_ = client.Remove(ctx, p.Serial, devicePath)
			return textResult(fmt.Sprintf("Recording saved to %s.", p.OutputPath)), nil
		},
	)
}

func registerLogcatTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "logcat",
			Description: "Get logcat output from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"filter":{"type":"string","description":"Logcat filter expression (e.g. ActivityManager:I *:S)."},"lines":{"type":"integer","description":"Number of recent lines to return."},"level":{"type":"string","description":"Minimum log level: V, D, I, W, E, F.","enum":["V","D","I","W","E","F"]},"format":{"type":"string","description":"Output format: brief, process, tag, thread, raw, time, threadtime, long.","enum":["brief","process","tag","thread","raw","time","threadtime","long"]}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Filter string
				Lines  int
				Level  string
				Format string
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			filter := p.Filter
			if p.Level != "" {
				if filter != "" {
					filter += " "
				}
				filter += "*:" + p.Level
			}
			opts := adb.LogcatOptions{
				Filter: filter,
				Format: p.Format,
				Count:  p.Lines,
			}
			entries, err := client.LogcatDump(ctx, p.Serial, opts)
			if err != nil {
				return nil, err
			}
			if len(entries) == 0 {
				return textResult("No log entries."), nil
			}
			var sb strings.Builder
			for _, e := range entries {
				fmt.Fprintf(&sb, "%s %s %s %s %s: %s\n",
					e.Timestamp.Format("01-02 15:04:05.000"),
					e.PID, e.TID, string(e.Level), e.Tag, e.Message)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "clear_logcat",
			Description: "Clear the logcat buffer on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.LogcatClear(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Logcat buffer cleared."), nil
		},
	)
}

func registerInputTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "tap",
			Description: "Tap on the screen at the given coordinates.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"x":{"type":"integer","description":"X coordinate."},"y":{"type":"integer","description":"Y coordinate."}},"required":["x","y"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				X      int
				Y      int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Tap(ctx, p.Serial, p.X, p.Y); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Tapped at (%d, %d).", p.X, p.Y)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "swipe",
			Description: "Perform a swipe gesture on the screen.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"x1":{"type":"integer","description":"Start X coordinate."},"y1":{"type":"integer","description":"Start Y coordinate."},"x2":{"type":"integer","description":"End X coordinate."},"y2":{"type":"integer","description":"End Y coordinate."},"duration_ms":{"type":"integer","description":"Swipe duration in milliseconds."}},"required":["x1","y1","x2","y2"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				X1         int `json:"x1"`
				Y1         int `json:"y1"`
				X2         int `json:"x2"`
				Y2         int `json:"y2"`
				DurationMS int `json:"duration_ms"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Swipe(ctx, p.Serial, p.X1, p.Y1, p.X2, p.Y2, p.DurationMS); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Swiped from (%d,%d) to (%d,%d).", p.X1, p.Y1, p.X2, p.Y2)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "key_event",
			Description: "Send a key event to the device (e.g. KEYCODE_HOME, KEYCODE_BACK, 3, 4).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"keycode":{"type":"string","description":"Key code name or number."}},"required":["keycode"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Keycode string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			keycode, err := strconv.Atoi(p.Keycode)
			if err != nil {
				// Map common keycode names to their integer values
				keycodeMap := map[string]int{
					"KEYCODE_HOME":            adb.KeyHome,
					"KEYCODE_BACK":            adb.KeyBack,
					"KEYCODE_CALL":            adb.KeyCall,
					"KEYCODE_ENDCALL":         adb.KeyEndCall,
					"KEYCODE_DPAD_UP":         adb.KeyDPadUp,
					"KEYCODE_DPAD_DOWN":       adb.KeyDPadDown,
					"KEYCODE_DPAD_LEFT":       adb.KeyDPadLeft,
					"KEYCODE_DPAD_RIGHT":      adb.KeyDPadRight,
					"KEYCODE_DPAD_CENTER":     adb.KeyDPadCenter,
					"KEYCODE_VOLUME_UP":       adb.KeyVolumeUp,
					"KEYCODE_VOLUME_DOWN":     adb.KeyVolumeDown,
					"KEYCODE_POWER":           adb.KeyPower,
					"KEYCODE_CAMERA":          adb.KeyCamera,
					"KEYCODE_CLEAR":           adb.KeyClear,
					"KEYCODE_MENU":            adb.KeyMenu,
					"KEYCODE_SEARCH":          adb.KeySearch,
					"KEYCODE_MEDIA_PLAY":      adb.KeyMediaPlay,
					"KEYCODE_MEDIA_STOP":      adb.KeyMediaStop,
					"KEYCODE_MEDIA_NEXT":      adb.KeyMediaNext,
					"KEYCODE_MEDIA_PREV":      adb.KeyMediaPrev,
					"KEYCODE_MUTE":            adb.KeyMute,
					"KEYCODE_TAB":             adb.KeyTab,
					"KEYCODE_ENTER":           adb.KeyEnter,
					"KEYCODE_DEL":             adb.KeyDelete,
					"KEYCODE_APP_SWITCH":      adb.KeyRecents,
					"KEYCODE_BRIGHTNESS_DOWN": adb.KeyBrightDown,
					"KEYCODE_BRIGHTNESS_UP":   adb.KeyBrightUp,
					"KEYCODE_SLEEP":           adb.KeySleep,
					"KEYCODE_WAKEUP":          adb.KeyWakeUp,
				}
				var ok bool
				keycode, ok = keycodeMap[strings.ToUpper(p.Keycode)]
				if !ok {
					return nil, fmt.Errorf("unknown keycode: %s", p.Keycode)
				}
			}
			if err := client.KeyEvent(ctx, p.Serial, keycode); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Sent key event: %s.", p.Keycode)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "input_text",
			Description: "Input text on the device as if typed on a keyboard.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"text":{"type":"string","description":"Text to input."}},"required":["text"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Text   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Text(ctx, p.Serial, p.Text); err != nil {
				return nil, err
			}
			return textResult("Text input sent."), nil
		},
	)
}

func registerPropertyTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "get_prop",
			Description: "Get a system property value from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"key":{"type":"string","description":"Property key (e.g. ro.product.model)."}},"required":["key"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Key    string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			value, err := client.GetProp(ctx, p.Serial, p.Key)
			if err != nil {
				return nil, err
			}
			return textResult(value), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_prop",
			Description: "Set a system property on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"key":{"type":"string","description":"Property key."},"value":{"type":"string","description":"Property value."}},"required":["key","value"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Key    string
				Value  string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetProp(ctx, p.Serial, p.Key, p.Value); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Set %s = %s.", p.Key, p.Value)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "list_props",
			Description: "List all system properties on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			props, err := client.ListProps(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			for k, v := range props {
				fmt.Fprintf(&sb, "[%s]: [%s]\n", k, v)
			}
			return textResult(sb.String()), nil
		},
	)
}

func registerForwardTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "forward",
			Description: "Set up port forwarding from host to device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"local":{"type":"string","description":"Local port specification (e.g. tcp:8080)."},"remote":{"type":"string","description":"Remote port specification (e.g. tcp:80)."}},"required":["local","remote"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Local  string
				Remote string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Forward(ctx, p.Serial, p.Local, p.Remote); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Forward %s -> %s established.", p.Local, p.Remote)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "forward_list",
			Description: "List all active port forwards.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			rules, err := client.ForwardList(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if len(rules) == 0 {
				return textResult("No active port forwards."), nil
			}
			var sb strings.Builder
			for _, r := range rules {
				fmt.Fprintf(&sb, "%s %s -> %s\n", r.Serial, r.Local, r.Remote)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "forward_remove",
			Description: "Remove a port forward.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"local":{"type":"string","description":"Local port specification to remove."}},"required":["local"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Local  string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.ForwardRemove(ctx, p.Serial, p.Local); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Removed forward for %s.", p.Local)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reverse",
			Description: "Set up reverse port forwarding from device to host.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"remote":{"type":"string","description":"Remote (device) port specification (e.g. tcp:8080)."},"local":{"type":"string","description":"Local (host) port specification (e.g. tcp:80)."}},"required":["remote","local"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Remote string
				Local  string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Reverse(ctx, p.Serial, p.Remote, p.Local); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Reverse %s -> %s established.", p.Remote, p.Local)), nil
		},
	)
}

func registerIntentTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "start_activity",
			Description: "Start an activity on the device using am start.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"component":{"type":"string","description":"Component name (e.g. com.example/.MainActivity)."},"action":{"type":"string","description":"Intent action (e.g. android.intent.action.VIEW)."},"data":{"type":"string","description":"Intent data URI."},"extras":{"type":"object","description":"Extra key-value pairs to pass.","additionalProperties":{"type":"string"}}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Component string
				Action    string
				Data      string
				Extras    map[string]string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			intent := adb.Intent{
				Action:    p.Action,
				Data:      p.Data,
				Component: p.Component,
				Extras:    p.Extras,
			}
			if err := client.StartActivity(ctx, p.Serial, intent); err != nil {
				return nil, err
			}
			return textResult("Activity started."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "send_broadcast",
			Description: "Send a broadcast intent on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"action":{"type":"string","description":"Broadcast action."},"component":{"type":"string","description":"Target component."},"extras":{"type":"object","description":"Extra key-value pairs.","additionalProperties":{"type":"string"}}},"required":["action"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Action    string
				Component string
				Extras    map[string]string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			intent := adb.Intent{
				Action:    p.Action,
				Component: p.Component,
				Extras:    p.Extras,
			}
			if err := client.SendBroadcast(ctx, p.Serial, intent); err != nil {
				return nil, err
			}
			return textResult("Broadcast sent."), nil
		},
	)
}

func registerSystemTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "reboot",
			Description: "Reboot the device. Mode can be normal, bootloader, or recovery.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"mode":{"type":"string","description":"Reboot mode: normal, bootloader, or recovery.","enum":["normal","bootloader","recovery"]}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Mode   string
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			mode := p.Mode
			if mode == "normal" {
				mode = ""
			}
			if err := client.Reboot(ctx, p.Serial, mode); err != nil {
				return nil, err
			}
			if p.Mode == "" || p.Mode == "normal" {
				return textResult("Device is rebooting."), nil
			}
			return textResult(fmt.Sprintf("Device is rebooting into %s.", p.Mode)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_battery",
			Description: "Get battery information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			info, err := client.GetBatteryInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Level: %d%%\nStatus: %s\nHealth: %s\nTemperature: %.1f°C\nVoltage: %dmV",
				info.Level, info.Status, info.Health, float64(info.Temperature)/10.0, info.Voltage)), nil
		},
	)
}

func registerConnectivityTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "wifi_control",
			Description: "Enable or disable WiFi on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string"},"enable":{"type":"boolean","description":"True to enable WiFi, false to disable."}},"required":["enable"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Enable bool
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if p.Enable {
				if err := client.EnableWifi(ctx, p.Serial); err != nil {
					return nil, err
				}
				return textResult("WiFi enabled."), nil
			}
			if err := client.DisableWifi(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("WiFi disabled."), nil
		},
	)
}
