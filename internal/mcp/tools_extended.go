package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alanisme/adb-tui/internal/adb"
)

func RegisterExtendedTools(s *Server, client *adb.Client) {
	registerBackupTools(s, client)
	registerSettingsTools(s, client)
	registerProcessTools(s, client)
	registerDumpsysTools(s, client)
	registerSecurityTools(s, client)
	registerTestingTools(s, client)
	registerExtendedShellTools(s, client)
	registerExtendedFileTools(s, client)
	registerExtendedNetworkTools(s, client)
	registerExtendedDeviceTools(s, client)
	registerClipboardTools(s, client)
	registerExtendedInputTools(s, client)
	registerExtendedForwardTools(s, client)
}

func registerBackupTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "bugreport",
			Description: "Generate a bug report from the device and save it to the specified path on the host.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"output_path":{"type":"string","description":"Output file path on the host."}},"required":["output_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				OutputPath string `json:"output_path"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Bugreport(ctx, p.Serial, p.OutputPath); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Bug report saved to %s.", p.OutputPath)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "sideload",
			Description: "Sideload an OTA package onto the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"ota_path":{"type":"string","description":"Path to the OTA package file."}},"required":["ota_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				OTAPath string `json:"ota_path"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Sideload(ctx, p.Serial, p.OTAPath); err != nil {
				return nil, err
			}
			return textResult("Sideload complete."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "backup",
			Description: "Backup device data to a file on the host.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"output_path":{"type":"string","description":"Output backup file path on the host."},"apk":{"type":"boolean","description":"Include APK files in backup."},"obb":{"type":"boolean","description":"Include OBB files in backup."},"shared":{"type":"boolean","description":"Include shared storage."},"all":{"type":"boolean","description":"Backup all installed apps."},"system":{"type":"boolean","description":"Include system apps."},"packages":{"type":"array","items":{"type":"string"},"description":"Specific package names to backup."}},"required":["output_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				OutputPath string   `json:"output_path"`
				APK        bool     `json:"apk"`
				OBB        bool     `json:"obb"`
				Shared     bool     `json:"shared"`
				All        bool     `json:"all"`
				System     bool     `json:"system"`
				Packages   []string `json:"packages"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			opts := adb.BackupOptions{
				APK:      p.APK,
				OBB:      p.OBB,
				Shared:   p.Shared,
				All:      p.All,
				System:   p.System,
				Packages: p.Packages,
			}
			if err := client.Backup(ctx, p.Serial, p.OutputPath, opts); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Backup saved to %s.", p.OutputPath)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "restore",
			Description: "Restore a backup file to the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"backup_path":{"type":"string","description":"Path to the backup file on the host."}},"required":["backup_path"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				BackupPath string `json:"backup_path"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Restore(ctx, p.Serial, p.BackupPath); err != nil {
				return nil, err
			}
			return textResult("Restore complete."), nil
		},
	)
}

func registerSettingsTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "get_setting",
			Description: "Get an Android setting value.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"namespace":{"type":"string","description":"Settings namespace.","enum":["system","secure","global"]},"key":{"type":"string","description":"Setting key name."}},"required":["namespace","key"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Namespace string `json:"namespace"`
				Key       string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			value, err := client.GetSetting(ctx, p.Serial, adb.SettingNamespace(p.Namespace), p.Key)
			if err != nil {
				return nil, err
			}
			return textResult(value), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "put_setting",
			Description: "Set an Android setting value.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"namespace":{"type":"string","description":"Settings namespace.","enum":["system","secure","global"]},"key":{"type":"string","description":"Setting key name."},"value":{"type":"string","description":"Setting value."}},"required":["namespace","key","value"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Namespace string `json:"namespace"`
				Key       string
				Value     string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.PutSetting(ctx, p.Serial, adb.SettingNamespace(p.Namespace), p.Key, p.Value); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Set %s/%s = %s.", p.Namespace, p.Key, p.Value)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "list_settings",
			Description: "List all settings in a namespace.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"namespace":{"type":"string","description":"Settings namespace.","enum":["system","secure","global"]}},"required":["namespace"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Namespace string `json:"namespace"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			settings, err := client.ListSettings(ctx, p.Serial, adb.SettingNamespace(p.Namespace))
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			for k, v := range settings {
				fmt.Fprintf(&sb, "%s=%s\n", k, v)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "delete_setting",
			Description: "Delete a setting from a namespace.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"namespace":{"type":"string","description":"Settings namespace.","enum":["system","secure","global"]},"key":{"type":"string","description":"Setting key to delete."}},"required":["namespace","key"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Namespace string `json:"namespace"`
				Key       string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.DeleteSetting(ctx, p.Serial, adb.SettingNamespace(p.Namespace), p.Key); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Deleted %s/%s.", p.Namespace, p.Key)), nil
		},
	)
}

func registerProcessTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "list_processes",
			Description: "List running processes on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			procs, err := client.ListProcesses(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%-8s %-12s %6s %6s %s\n", "PID", "USER", "CPU%", "MEM%", "NAME")
			for _, proc := range procs {
				fmt.Fprintf(&sb, "%-8d %-12s %6.1f %6.1f %s\n",
					proc.PID, proc.User, proc.CPU, proc.MEM, proc.Name)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "kill_process",
			Description: "Kill a process by PID on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"pid":{"type":"integer","description":"Process ID to kill."}},"required":["pid"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				PID    int `json:"pid"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.KillProcess(ctx, p.Serial, p.PID); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Killed process %d.", p.PID)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "kill_process_by_name",
			Description: "Kill all processes matching a name on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"name":{"type":"string","description":"Process name to kill."}},"required":["name"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Name   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.KillProcessByName(ctx, p.Serial, p.Name); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Killed processes matching %s.", p.Name)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "memory_info",
			Description: "Get system memory information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			info, err := client.GetMemInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Total: %d kB\nFree: %d kB\nAvailable: %d kB",
				info.Total, info.Free, info.Available)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "app_memory_info",
			Description: "Get memory usage for a specific application.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"package":{"type":"string","description":"Package name of the application."}},"required":["package"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Package string `json:"package"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			output, err := client.GetAppMemInfo(ctx, p.Serial, p.Package)
			if err != nil {
				return nil, err
			}
			return textResult(output), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "top_processes",
			Description: "Get top processes sorted by resource usage.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"count":{"type":"integer","description":"Number of top processes to return."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Count  int
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			count := p.Count
			if count <= 0 {
				count = 10
			}
			procs, err := client.GetTopProcesses(ctx, p.Serial, count)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%-8s %-12s %6s %6s %s\n", "PID", "USER", "CPU%", "MEM%", "NAME")
			for _, proc := range procs {
				fmt.Fprintf(&sb, "%-8d %-12s %6.1f %6.1f %s\n",
					proc.PID, proc.User, proc.CPU, proc.MEM, proc.Name)
			}
			return textResult(sb.String()), nil
		},
	)
}

func registerDumpsysTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "dumpsys",
			Description: "Run dumpsys for a specific system service.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"service":{"type":"string","description":"System service name (e.g. activity, battery, window)."}},"required":["service"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial  string
				Service string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			output, err := client.Dumpsys(ctx, p.Serial, p.Service)
			if err != nil {
				return nil, err
			}
			return textResult(output), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "dumpsys_list",
			Description: "List all available dumpsys services.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			services, err := client.DumpsysList(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(strings.Join(services, "\n")), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "battery_info",
			Description: "Get detailed battery information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
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

	s.RegisterTool(
		Tool{
			Name:        "display_info",
			Description: "Get display configuration information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			info, err := client.GetDisplayInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Resolution: %dx%d\nDensity: %d dpi\nFPS: %.1f",
				info.Width, info.Height, info.Density, info.FPS)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "window_info",
			Description: "Get window hierarchy information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			output, err := client.GetWindowInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(output), nil
		},
	)
}

func registerSecurityTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "selinux_status",
			Description: "Get SELinux enforcement status on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			status, err := client.GetSELinuxStatus(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(status), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_selinux",
			Description: "Set SELinux enforcement mode on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"mode":{"type":"string","description":"SELinux mode.","enum":["enforcing","permissive"]}},"required":["mode"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Mode   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetSELinux(ctx, p.Serial, p.Mode); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("SELinux set to %s.", p.Mode)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "list_permissions",
			Description: "List system permissions on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"group":{"type":"string","description":"Permission group to filter by. If empty, lists all permissions."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Group  string
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			perms, err := client.ListPermissions(ctx, p.Serial, p.Group)
			if err != nil {
				return nil, err
			}
			return textResult(strings.Join(perms, "\n")), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_apk_path",
			Description: "Get the installed APK file path for a package.",
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
			path, err := client.GetAPKPath(ctx, p.Serial, p.Package)
			if err != nil {
				return nil, err
			}
			return textResult(path), nil
		},
	)
}

func registerTestingTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "run_monkey",
			Description: "Run monkey stress test on an application.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"package":{"type":"string","description":"Target package name."},"events":{"type":"integer","description":"Number of random events to generate."},"seed":{"type":"integer","description":"Seed for pseudo-random number generator."},"throttle":{"type":"integer","description":"Delay between events in milliseconds."}},"required":["package","events"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial   string
				Package  string `json:"package"`
				Events   int
				Seed     int
				Throttle int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			opts := adb.MonkeyOptions{
				Seed:     int64(p.Seed),
				Throttle: p.Throttle,
			}
			result, err := client.RunMonkey(ctx, p.Serial, p.Package, p.Events, opts)
			if err != nil {
				if result != nil {
					return textResult(result.Output), nil
				}
				return nil, err
			}
			return textResult(result.Output), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "run_instrumentation",
			Description: "Run an instrumentation test on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"component":{"type":"string","description":"Instrumentation component (e.g. com.example.test/androidx.test.runner.AndroidJUnitRunner)."},"runner":{"type":"string","description":"Test runner class. Overrides the runner in component if set."},"arguments":{"type":"object","description":"Key-value arguments to pass to the instrumentation.","additionalProperties":{"type":"string"}},"raw_output":{"type":"boolean","description":"Use raw output mode."}},"required":["component"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial    string
				Component string
				Runner    string
				Arguments map[string]string
				RawOutput bool `json:"raw_output"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			opts := adb.InstrumentOptions{
				Runner:    p.Runner,
				Arguments: p.Arguments,
				RawOutput: p.RawOutput,
			}
			result, err := client.RunInstrumentation(ctx, p.Serial, p.Component, opts)
			if err != nil {
				if result != nil {
					return textResult(result.Output), nil
				}
				return nil, err
			}
			return textResult(result.Output), nil
		},
	)
}

func registerExtendedShellTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "root",
			Description: "Restart ADB daemon with root permissions.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			result, err := client.ExecDevice(ctx, p.Serial, "root")
			if err != nil {
				return nil, err
			}
			return textResult(result.Output), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "unroot",
			Description: "Restart ADB daemon without root permissions.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			result, err := client.ExecDevice(ctx, p.Serial, "unroot")
			if err != nil {
				return nil, err
			}
			return textResult(result.Output), nil
		},
	)
}

func registerExtendedFileTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "chmod",
			Description: "Change file permissions on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."},"mode":{"type":"string","description":"Permission mode (e.g. 755, 644)."}},"required":["path","mode"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
				Mode   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Chmod(ctx, p.Serial, p.Path, p.Mode); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Changed permissions of %s to %s.", p.Path, p.Mode)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "chown",
			Description: "Change file owner on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"File path on the device."},"owner":{"type":"string","description":"Owner specification (e.g. root:root, system:system)."}},"required":["path","owner"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
				Owner  string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.Chown(ctx, p.Serial, p.Path, p.Owner); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Changed owner of %s to %s.", p.Path, p.Owner)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "find_files",
			Description: "Search for files by name on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"path":{"type":"string","description":"Directory to search in."},"name":{"type":"string","description":"File name pattern to search for."}},"required":["path","name"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Path   string
				Name   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			files, err := client.Find(ctx, p.Serial, p.Path, p.Name)
			if err != nil {
				return nil, err
			}
			if len(files) == 0 {
				return textResult("No files found."), nil
			}
			return textResult(strings.Join(files, "\n")), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "disk_usage",
			Description: "Get disk usage information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			disks, err := client.DiskFree(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%-20s %8s %8s %8s %6s %s\n",
				"Filesystem", "Size", "Used", "Avail", "Use%", "Mount")
			for _, d := range disks {
				fmt.Fprintf(&sb, "%-20s %8s %8s %8s %6s %s\n",
					d.Filesystem, d.Size, d.Used, d.Available, d.UsePercent, d.MountPoint)
			}
			return textResult(sb.String()), nil
		},
	)
}

func registerExtendedNetworkTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "netstat",
			Description: "List network connections on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			conns, err := client.GetNetstat(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if len(conns) == 0 {
				return textResult("No network connections."), nil
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%-8s %-25s %-25s %s\n", "Proto", "Local", "Remote", "State")
			for _, c := range conns {
				fmt.Fprintf(&sb, "%-8s %-25s %-25s %s\n",
					c.Protocol, c.LocalAddr, c.RemoteAddr, c.State)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "ping_host",
			Description: "Ping a host from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"host":{"type":"string","description":"Host to ping."},"count":{"type":"integer","description":"Number of ping packets to send."}},"required":["host"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Host   string
				Count  int
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			count := p.Count
			if count <= 0 {
				count = 4
			}
			result, err := client.Ping(ctx, p.Serial, p.Host, count)
			if err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Transmitted: %d\nReceived: %d\nLoss: %.1f%%\nAvg RTT: %.2f ms",
				result.Transmitted, result.Received, result.LossPercent, result.AvgRTT)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_ip_address",
			Description: "Get the IP address of the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			ip, err := client.GetIPAddress(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(ip), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "get_network_info",
			Description: "Get network interface information from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			info, err := client.GetNetworkInfo(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			for _, iface := range info.Interfaces {
				fmt.Fprintf(&sb, "%s: %s/%s\n", iface.Name, iface.IPAddress, iface.Mask)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "tcpip_mode",
			Description: "Switch device to TCP/IP mode for wireless debugging.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"port":{"type":"integer","description":"TCP port to listen on. Defaults to 5555."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Port   int
			}
			if params != nil {
				json.Unmarshal(params, &p)
			}
			port := p.Port
			if port <= 0 {
				port = 5555
			}
			if err := client.TcpIp(ctx, p.Serial, port); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Device switched to TCP/IP mode on port %d.", port)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "usb_mode",
			Description: "Switch device back to USB mode.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.Usb(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Device switched to USB mode."), nil
		},
	)
}

func registerExtendedDeviceTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "get_device_state",
			Description: "Get the current state of a device (device, offline, unauthorized, etc.).",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			state, err := client.GetState(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			return textResult(string(state)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "is_rooted",
			Description: "Check if the device has root access.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			rooted := client.IsRooted(ctx, p.Serial)
			if rooted {
				return textResult("Device has root access."), nil
			}
			return textResult("Device does not have root access."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "remount",
			Description: "Remount device filesystem partitions as read-write. Requires root.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.Remount(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("Filesystem remounted."), nil
		},
	)
}

func registerClipboardTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "get_clipboard",
			Description: "Get the current clipboard content from the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			text, err := client.GetClipboard(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if text == "" {
				return textResult("Clipboard is empty."), nil
			}
			return textResult(text), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "set_clipboard",
			Description: "Set the clipboard content on the device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"text":{"type":"string","description":"Text to copy to clipboard."}},"required":["text"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial string
				Text   string
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			if err := client.SetClipboard(ctx, p.Serial, p.Text); err != nil {
				return nil, err
			}
			return textResult("Clipboard set."), nil
		},
	)
}

func registerExtendedInputTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "long_press",
			Description: "Perform a long press at the given coordinates.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"x":{"type":"integer","description":"X coordinate."},"y":{"type":"integer","description":"Y coordinate."},"duration_ms":{"type":"integer","description":"Press duration in milliseconds. Defaults to 1000."}},"required":["x","y"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial     string
				X          int
				Y          int
				DurationMS int `json:"duration_ms"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			dur := p.DurationMS
			if dur <= 0 {
				dur = 1000
			}
			if err := client.LongPress(ctx, p.Serial, p.X, p.Y, dur); err != nil {
				return nil, err
			}
			return textResult(fmt.Sprintf("Long pressed at (%d, %d) for %dms.", p.X, p.Y, dur)), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "human_swipe",
			Description: "Perform a natural swipe gesture with slight random variation to simulate human touch.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."},"screen_width":{"type":"integer","description":"Screen width in pixels."},"screen_height":{"type":"integer","description":"Screen height in pixels."},"start_x":{"type":"number","description":"Start X as fraction of screen width (0.0-1.0)."},"start_y":{"type":"number","description":"Start Y as fraction of screen height (0.0-1.0)."},"end_x":{"type":"number","description":"End X as fraction of screen width (0.0-1.0)."},"end_y":{"type":"number","description":"End Y as fraction of screen height (0.0-1.0)."},"duration_ms":{"type":"integer","description":"Swipe duration in milliseconds."}},"required":["screen_width","screen_height","start_x","start_y","end_x","end_y"]}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct {
				Serial       string
				ScreenWidth  int     `json:"screen_width"`
				ScreenHeight int     `json:"screen_height"`
				StartX       float64 `json:"start_x"`
				StartY       float64 `json:"start_y"`
				EndX         float64 `json:"end_x"`
				EndY         float64 `json:"end_y"`
				DurationMS   int     `json:"duration_ms"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
			dur := p.DurationMS
			if dur <= 0 {
				dur = 300
			}
			gesture := adb.GestureParams{
				StartXFrac: p.StartX,
				StartYFrac: p.StartY,
				EndXFrac:   p.EndX,
				EndYFrac:   p.EndY,
				DurationMs: dur,
			}
			if err := client.HumanSwipe(ctx, p.Serial, p.ScreenWidth, p.ScreenHeight, gesture); err != nil {
				return nil, err
			}
			return textResult("Human swipe performed."), nil
		},
	)
}

func registerExtendedForwardTools(s *Server, client *adb.Client) {
	s.RegisterTool(
		Tool{
			Name:        "forward_remove_all",
			Description: "Remove all port forwards for a device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ForwardRemoveAll(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("All port forwards removed."), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reverse_list",
			Description: "List all active reverse port forwards.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			rules, err := client.ReverseList(ctx, p.Serial)
			if err != nil {
				return nil, err
			}
			if len(rules) == 0 {
				return textResult("No active reverse forwards."), nil
			}
			var sb strings.Builder
			for _, r := range rules {
				fmt.Fprintf(&sb, "%s %s -> %s\n", r.Serial, r.Remote, r.Local)
			}
			return textResult(sb.String()), nil
		},
	)

	s.RegisterTool(
		Tool{
			Name:        "reverse_remove_all",
			Description: "Remove all reverse port forwards for a device.",
			InputSchema: schema(`{"type":"object","properties":{"serial":{"type":"string","description":"Device serial number."}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Serial string }
			if params != nil {
				json.Unmarshal(params, &p)
			}
			if err := client.ReverseRemoveAll(ctx, p.Serial); err != nil {
				return nil, err
			}
			return textResult("All reverse forwards removed."), nil
		},
	)
}
