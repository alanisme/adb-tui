<p align="center">
  <h1 align="center">adb-tui</h1>
  <p align="center">
    <strong>Android 遇上 AI，尽在终端之中。</strong>
  </p>
  <p align="center">
    全功能 Android 调试桥终端界面 + MCP 服务器。<br>
    用键盘控制任何 Android 设备，或者让 AI 帮你完成。
  </p>
  <p align="center">
    <a href="https://github.com/alanisme/adb-tui/actions/workflows/ci.yml"><img src="https://github.com/alanisme/adb-tui/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/alanisme/adb-tui/releases/latest"><img src="https://img.shields.io/github/v/release/alanisme/adb-tui" alt="Release"></a>
    <img src="https://img.shields.io/github/go-mod/go-version/alanisme/adb-tui" alt="Go Version">
    <a href="../LICENSE"><img src="https://img.shields.io/github/license/alanisme/adb-tui" alt="License"></a>
  </p>
  <p align="center">
    <a href="#快速开始">快速开始</a> &middot;
    <a href="#功能视图">功能视图</a> &middot;
    <a href="#mcp-服务器与-ai-集成">MCP + AI</a> &middot;
    <a href="../README.md">English</a>
  </p>
</p>

---

## 什么是 adb-tui？

**adb-tui** 是一个基于终端的 ADB 全功能控制台。不再需要记忆命令和解析原始输出，11 个专用视图，每个只需一个按键即可切换。

同时内置 **MCP 服务器**，暴露 **120+ 个工具**，AI 智能体（如 Claude）可以自主管理 Android 设备：安装应用、读取日志、截图、控制输入、管理文件，全部通过结构化的工具调用完成。

### 核心亮点

- **完整的 ADB 覆盖**：设备、Shell、Logcat、文件、应用、进程、端口、输入控制、设置、性能监控
- **AI 原生**：内置 MCP 服务器（stdio + HTTP/SSE），将 Android 设备变为 AI 可控端点
- **实时 Logcat**：流式传输、按级别/标签/PID 过滤、搜索、导出，轻松处理 10000+ 条日志
- **无线调试**：一键切换 USB 和 TCP/IP 模式
- **安全设计**：所有 Shell 命令使用参数化执行，杜绝注入风险
- **全平台兼容**：macOS、Linux、Windows；Samsung、Pixel、小米、一加、模拟器，任何 ADB 兼容设备
- **零配置**：安装即用，无需额外设置

## 安装

### macOS (Homebrew)

```bash
brew install alanisme/tap/adb-tui
```

### Linux / macOS (脚本)

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

### 从源码构建

```bash
git clone https://github.com/alanisme/adb-tui.git
cd adb-tui
make build
```

预编译二进制文件也可在 [Releases](https://github.com/alanisme/adb-tui/releases) 页面下载。

> **前置依赖：** ADB 已安装并在 PATH 中。

## 功能视图

按数字键即可跳转到任意视图。

| 按键 | 视图 | 功能 |
|------|------|------|
| `1` | **设备** | 连接、断开、无线切换、ADB 版本显示 |
| `2` | **信息** | 硬件规格、电池、存储、网络、CPU 一览 |
| `3` | **Shell** | 交互式 Shell，支持命令历史 |
| `4` | **Logcat** | 实时日志流，级别/标签/搜索过滤，支持导出 |
| `5` | **文件** | 浏览文件系统，推送/拉取文件，管理权限 |
| `6` | **应用** | 安装/卸载 APK，管理权限，查看 APK 路径 |
| `7` | **端口转发** | TCP 正向和反向端口规则管理 |
| `8` | **输入控制** | 点击、滑动、长按、按键事件、文本输入 |
| `9` | **进程** | 进程列表，按 PID 或名称终止，查看内存信息 |
| `0` | **设置** | 读写 system/secure/global 设置项 |
| `-` | **性能** | 实时 CPU、内存、电池、显示指标 |

`?` 帮助 &middot; `/` 搜索 &middot; `Tab` 切换视图 &middot; `j/k` 导航 &middot; `T` 主题 &middot; `q` 退出（需确认） &middot; `^c` 退出

## MCP 服务器与 AI 集成

**adb-tui** 实现了 [Model Context Protocol](https://modelcontextprotocol.io)，将 Android 设备变为 AI 智能体的工具端点。

这意味着 Claude 或任何 MCP 兼容客户端可以：
- 安装和管理应用
- 读取和过滤 Logcat 日志
- 推送/拉取文件
- 截图和录屏
- 控制触摸输入和按键事件
- 查询设备信息、电池、网络状态
- 管理系统设置和权限

### 配置 Claude Desktop

在 `claude_desktop_config.json` 中添加：

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

### HTTP/SSE 传输模式

适用于远程访问或多客户端场景：

```bash
adb-tui mcp --transport http --addr :8080
```

### 全部 120+ 个 MCP 工具

<details>
<summary>展开完整工具列表</summary>

| 分类 | 工具 |
|------|------|
| **设备** | `list_devices` `device_info` `connect` `disconnect` `get_device_state` `is_rooted` `remount` |
| **Shell** | `shell` `root` `unroot` |
| **应用** | `list_packages` `install_apk` `uninstall_package` `package_info` `force_stop` `clear_data` `enable_package` `disable_package` `grant_permission` `revoke_permission` |
| **文件** | `push_file` `pull_file` `list_files` `read_file` `write_file` `delete_file` `mkdir` `stat_file` `find_files` `disk_usage` `chmod` `chown` |
| **屏幕** | `screenshot` `screen_record` `get_screen_size` |
| **日志** | `logcat` `clear_logcat` |
| **输入** | `tap` `swipe` `key_event` `input_text` `long_press` `human_swipe` |
| **UI 自动化** | `get_ui_hierarchy` `find_element` `tap_element` `wait_for_element` `get_focused_app` `get_current_activity` |
| **属性** | `get_prop` `set_prop` `list_props` |
| **端口转发** | `forward` `forward_list` `forward_remove` `forward_remove_all` `reverse` `reverse_list` `reverse_remove_all` |
| **Intent** | `start_activity` `send_broadcast` `open_url` |
| **Activity** | `get_current_activity` `list_activities` |
| **系统** | `reboot` `get_battery` |
| **连接** | `wifi_control` `get_ip_address` `get_network_info` `tcpip_mode` `usb_mode` `ping_host` |
| **设置** | `get_setting` `put_setting` `list_settings` `delete_setting` |
| **设备控制** | `screen_on` `screen_off` `is_screen_on` `get_brightness` `set_brightness` `set_rotation` `set_auto_rotation` `get_airplane_mode` `set_airplane_mode` |
| **音量与媒体** | `volume_up` `volume_down` `volume_mute` `media_play` `media_pause` `media_next` `media_previous` |
| **显示** | `set_display_size` `reset_display_size` `set_density` `reset_density` `get_font_scale` `set_font_scale` |
| **通知** | `list_notifications` `expand_notifications` `collapse_notifications` `expand_quick_settings` |
| **进程** | `list_processes` `kill_process` `kill_process_by_name` `top_processes` `memory_info` `app_memory_info` |
| **Dumpsys** | `dumpsys` `dumpsys_list` `battery_info` `display_info` `window_info` |
| **安全** | `selinux_status` `set_selinux` `list_permissions` `get_apk_path` |
| **测试** | `run_monkey` `run_instrumentation` |
| **电池模拟** | `simulate_battery` `reset_battery` |
| **备份** | `bugreport` `sideload` `backup` `restore` |
| **网络** | `netstat` |
| **剪贴板** | `get_clipboard` `set_clipboard` |

</details>

## CLI 命令

无需启动 TUI，直接作为脚本化 CLI 使用：

```bash
adb-tui devices                           # 列出已连接设备
adb-tui shell <serial> <command>          # 执行 Shell 命令
adb-tui screenshot <serial> output.png    # 截图
adb-tui install <serial> app.apk         # 安装 APK
adb-tui mcp                              # 启动 MCP 服务器（stdio）
adb-tui version                          # 查看版本信息
```

## 配置

可选。放置在 `~/.config/adb-tui/config.json`：

```json
{
  "adb_path": "/usr/local/bin/adb",
  "theme": "default"
}
```

可用主题：`default`、`nord`、`tokyonight`、`catppuccin`。也可以在应用内按 `T` 切换主题。

## 使用场景

- **Android 应用调试**：实时查看 logcat、检查 UI 层级、切换系统设置，无需离开终端
- **设备批量管理**：同时连接多台设备，一个面板查看电量/存储/网络状态
- **自动化测试**：运行 monkey 测试、通过 `tap_element`/`find_element` 驱动 UI、自动截图
- **AI 驱动的设备控制**：搭配 Claude 或任何 MCP 客户端，构建自主 Android 工作流
- **远程设备检查**：无线 ADB + HTTP/SSE transport，适用于无头环境和 CI
- **逆向工程**：浏览文件系统、dump 窗口信息、查看运行中的进程和权限

## 项目结构

```
cmd/adb-tui/       CLI 入口（cobra）
internal/adb/       ADB 客户端库，参数化执行、并发安全、有测试覆盖
internal/tui/       Bubbletea TUI，11 个视图，Elm 架构
internal/mcp/       MCP 服务器，120+ 个工具，stdio + HTTP/SSE 传输
internal/config/    配置管理
pkg/jsonrpc/        JSON-RPC 2.0 实现
```

## 贡献

欢迎贡献。提交较大改动前请先开 Issue 讨论。

```bash
make check    # 格式化、vet、lint、带竞态检测的测试
make ci       # 完整 CI 流水线（含覆盖率门槛）
```

## 许可证

[Apache-2.0](../LICENSE) &copy; [Alan](mailto:alanisme.my@gmail.com)
