<p align="center">
  <h1 align="center">adb-tui</h1>
  <p align="center">
    <strong>Android meets AI. From your terminal.</strong>
  </p>
  <p align="center">
    A full-featured terminal UI and MCP server for Android Debug Bridge.<br>
    Control any Android device with keystrokes, or let AI do it for you.
  </p>
  <p align="center">
    <a href="https://github.com/alanisme/adb-tui/actions/workflows/ci.yml"><img src="https://github.com/alanisme/adb-tui/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/alanisme/adb-tui/releases/latest"><img src="https://img.shields.io/github/v/release/alanisme/adb-tui" alt="Release"></a>
    <img src="https://img.shields.io/github/go-mod/go-version/alanisme/adb-tui" alt="Go Version">
    <a href="LICENSE"><img src="https://img.shields.io/github/license/alanisme/adb-tui" alt="License"></a>
  </p>
  <p align="center">
    <a href="#install">Install</a> &middot;
    <a href="#tui-views">Views</a> &middot;
    <a href="#mcp-server--ai-integration">MCP + AI</a> &middot;
    <a href="#cli-commands">CLI</a> &middot;
    <a href="docs/README_zh.md">中文文档</a>
  </p>
</p>

---

**adb-tui** gives you a complete Android debugging dashboard in one terminal window. 11 interactive views, each one keystroke away. No commands to memorize, no output to parse.

It also doubles as an **MCP server with 120+ tools**, turning any Android device into an AI-controllable endpoint. Claude, or any MCP-compatible agent, can install apps, stream logs, capture screens, manage files, and more, all autonomously.

### Highlights

- **Full ADB coverage**: devices, shell, logcat, files, packages, processes, ports, input, settings, performance
- **AI-native**: built-in MCP server (stdio + HTTP/SSE) for Claude and other AI agents
- **Real-time logcat**: stream, filter by level/tag/PID, search, export, handles 10k+ entries
- **Wireless debugging**: one-key switch between USB and TCP/IP
- **Secure**: all shell commands use parameterized execution, no injection vectors
- **Cross-platform**: macOS, Linux, Windows. Any ADB-compatible Android device or emulator
- **Zero config**: install and run

## Install

### macOS (Homebrew)

```bash
brew install alanisme/tap/adb-tui
```

### Linux / macOS (script)

```bash
curl -fsSL https://raw.githubusercontent.com/alanisme/adb-tui/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/alanisme/adb-tui/main/install.ps1 | iex
```

### Go

```bash
go install github.com/alanisme/adb-tui/cmd/adb-tui@latest
```

### From source

```bash
git clone https://github.com/alanisme/adb-tui.git
cd adb-tui
make build
```

### GitHub Releases

Pre-built binaries for all platforms available on the [Releases](https://github.com/alanisme/adb-tui/releases) page.

> **Prerequisite:** ADB must be installed and in your PATH.

## TUI Views

Press a number key to jump to any view instantly.

| Key | View | What you can do |
|-----|------|-----------------|
| `1` | **Devices** | Connect, disconnect, wireless toggle, ADB version display |
| `2` | **Info** | Hardware specs, battery, storage, network, CPU at a glance |
| `3` | **Shell** | Interactive shell with command history |
| `4` | **Logcat** | Real-time log streaming with level/tag/search filters and export |
| `5` | **Files** | Browse filesystem, push/pull files, manage permissions |
| `6` | **Packages** | Install/uninstall APKs, manage permissions, view APK paths |
| `7` | **Port Forward** | TCP forward and reverse port rules |
| `8` | **Input** | Tap, swipe, long press, key events, text injection |
| `9` | **Processes** | Process list, kill by PID or name, per-app memory info |
| `0` | **Settings** | Read/write system, secure, and global settings |
| `-` | **Performance** | Live CPU, memory, battery, and display metrics |

`?` help &middot; `/` search &middot; `Tab` cycle views &middot; `j/k` navigate &middot; `T` theme &middot; `q` quit (confirm) &middot; `^c` quit

## MCP Server & AI Integration

**adb-tui** implements the [Model Context Protocol](https://modelcontextprotocol.io), turning your Android device into a tool-equipped endpoint for AI agents.

This means Claude, or any MCP-compatible client, can:
- Install and manage apps
- Read and filter logcat output
- Push/pull files, take screenshots, record screen
- Control touch input and key events
- Query device info, battery, network status
- Manage system settings and permissions

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "adb": {
      "command": "adb-tui",
      "args": ["mcp"]
    }
  }
}
```

### HTTP/SSE Transport

```bash
adb-tui mcp --transport http --addr :8080
```

### All 120+ MCP Tools

<details>
<summary>Expand full tool list</summary>

| Category | Tools |
|----------|-------|
| **Devices** | `list_devices` `device_info` `connect` `disconnect` `get_device_state` `is_rooted` `remount` |
| **Shell** | `shell` `root` `unroot` |
| **Packages** | `list_packages` `install_apk` `uninstall_package` `package_info` `force_stop` `clear_data` `enable_package` `disable_package` `grant_permission` `revoke_permission` |
| **Files** | `push_file` `pull_file` `list_files` `read_file` `write_file` `delete_file` `mkdir` `stat_file` `find_files` `disk_usage` `chmod` `chown` |
| **Screen** | `screenshot` `screen_record` `get_screen_size` |
| **Logcat** | `logcat` `clear_logcat` |
| **Input** | `tap` `swipe` `key_event` `input_text` `long_press` `human_swipe` |
| **UI Automation** | `get_ui_hierarchy` `find_element` `tap_element` `wait_for_element` `get_focused_app` `get_current_activity` |
| **Properties** | `get_prop` `set_prop` `list_props` |
| **Port Forward** | `forward` `forward_list` `forward_remove` `forward_remove_all` `reverse` `reverse_list` `reverse_remove_all` |
| **Intent** | `start_activity` `send_broadcast` `open_url` |
| **Activity** | `get_current_activity` `list_activities` |
| **System** | `reboot` `get_battery` |
| **Connectivity** | `wifi_control` `get_ip_address` `get_network_info` `tcpip_mode` `usb_mode` `ping_host` |
| **Settings** | `get_setting` `put_setting` `list_settings` `delete_setting` |
| **Device Control** | `screen_on` `screen_off` `is_screen_on` `get_brightness` `set_brightness` `set_rotation` `set_auto_rotation` `get_airplane_mode` `set_airplane_mode` |
| **Volume & Media** | `volume_up` `volume_down` `volume_mute` `media_play` `media_pause` `media_next` `media_previous` |
| **Display** | `set_display_size` `reset_display_size` `set_density` `reset_density` `get_font_scale` `set_font_scale` |
| **Notifications** | `list_notifications` `expand_notifications` `collapse_notifications` `expand_quick_settings` |
| **Process** | `list_processes` `kill_process` `kill_process_by_name` `top_processes` `memory_info` `app_memory_info` |
| **Dumpsys** | `dumpsys` `dumpsys_list` `battery_info` `display_info` `window_info` |
| **Security** | `selinux_status` `set_selinux` `list_permissions` `get_apk_path` |
| **Testing** | `run_monkey` `run_instrumentation` |
| **Battery Sim** | `simulate_battery` `reset_battery` |
| **Backup** | `bugreport` `sideload` `backup` `restore` |
| **Network** | `netstat` |
| **Clipboard** | `get_clipboard` `set_clipboard` |

</details>

## CLI Commands

Use adb-tui as a scriptable CLI without launching the TUI:

```bash
adb-tui devices                           # List connected devices
adb-tui shell <serial> <command>          # Run shell command
adb-tui screenshot <serial> output.png    # Capture screenshot
adb-tui install <serial> app.apk         # Install APK
adb-tui mcp                              # Start MCP server (stdio)
adb-tui version                          # Print version info
```

## Configuration

Optional. Place at `~/.config/adb-tui/config.json`:

```json
{
  "adb_path": "/usr/local/bin/adb",
  "theme": "default"
}
```

Available themes: `default`, `nord`, `tokyonight`, `catppuccin`. You can also switch themes in-app by pressing `T`.

## Use Cases

- **Android app debugging**: stream logcat, inspect UI hierarchy, toggle settings without leaving the terminal
- **Device fleet management**: connect multiple devices, query battery/storage/network in one dashboard
- **Automated testing**: run monkey tests, drive UI with `tap_element`/`find_element`, capture screenshots
- **AI-powered device control**: pair with Claude or any MCP client to build autonomous Android workflows
- **Remote device inspection**: wireless ADB + HTTP/SSE transport for headless or CI environments
- **Reverse engineering**: browse filesystem, dump window info, inspect running processes and permissions

## Architecture

```
cmd/adb-tui/       CLI entry point (cobra)
internal/adb/       ADB client library, parameterized, concurrent, tested
internal/tui/       Bubbletea TUI, 11 views, Elm architecture
internal/mcp/       MCP server, 120+ tools, stdio + HTTP/SSE transport
internal/config/    Configuration management
pkg/jsonrpc/        JSON-RPC 2.0 implementation
```

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
make check    # fmt, vet, lint, test with race detector
make ci       # full CI pipeline with coverage gate
```

## License

[Apache-2.0](LICENSE) &copy; [Alan](mailto:alanisme.my@gmail.com)
