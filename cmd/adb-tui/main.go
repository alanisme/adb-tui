package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alanisme/adb-tui/internal/adb"
	"github.com/alanisme/adb-tui/internal/config"
	"github.com/alanisme/adb-tui/internal/mcp"
	"github.com/alanisme/adb-tui/internal/mcp/transport"
	"github.com/alanisme/adb-tui/internal/tui"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	// Restore terminal on unexpected panic so the shell isn't left in alt-screen mode.
	defer func() {
		if r := recover(); r != nil {
			// Best-effort: leave alt screen and show cursor
			fmt.Fprint(os.Stderr, "\x1b[?1049l\x1b[?25h")
			fmt.Fprintf(os.Stderr, "panic: %v\n", r)
			os.Exit(2)
		}
	}()

	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var adbPath string

	root := &cobra.Command{
		Use:          "adb-tui",
		Short:        "Terminal UI for Android Debug Bridge",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(adbPath)
			if err != nil {
				return err
			}
			// Load saved theme from config
			if cfg, _ := config.Load(); cfg.Theme != "" {
				if t, ok := tui.BuiltinThemes[cfg.Theme]; ok {
					tui.ApplyTheme(t)
				}
			}
			return tui.NewApp(client).Run()
		},
	}

	root.PersistentFlags().StringVar(&adbPath, "adb-path", "", "path to adb binary")

	root.AddCommand(
		mcpCmd(&adbPath),
		versionCmd(),
		devicesCmd(&adbPath),
		shellCmd(&adbPath),
		screenshotCmd(&adbPath),
		installCmd(&adbPath),
		completionCmd(),
	)

	return root
}

func mcpCmd(adbPath *string) *cobra.Command {
	var (
		transportFlag string
		addr          string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(*adbPath)
			if err != nil {
				return err
			}

			srv := mcp.NewServer("adb-tui", Version)
			mcp.RegisterADBTools(srv, client)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			switch transportFlag {
			case "stdio":
				t := transport.NewStdioTransport()
				return t.Serve(ctx, srv)
			case "http":
				fmt.Fprintf(os.Stderr, "MCP HTTP server listening on %s\n", addr)
				t := transport.NewHTTPTransport(addr)
				return t.Serve(ctx, srv)
			default:
				return fmt.Errorf("unsupported transport: %s", transportFlag)
			}
		},
	}

	cmd.Flags().StringVar(&transportFlag, "transport", "stdio", "transport type (stdio|http)")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address for HTTP transport")

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("adb-tui %s\n", Version)
			fmt.Printf("commit: %s\n", Commit)
			fmt.Printf("built:  %s\n", BuildDate)
		},
	}
}

func devicesCmd(adbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List connected devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(*adbPath)
			if err != nil {
				return err
			}
			devices, err := client.ListDevices(cmd.Context())
			if err != nil {
				return err
			}
			if len(devices) == 0 {
				fmt.Println("No devices connected.")
				return nil
			}
			for _, d := range devices {
				fmt.Printf("%s\t%s\tmodel:%s\tproduct:%s\n",
					d.Serial, d.State, d.Model, d.Product)
			}
			return nil
		},
	}
}

func shellCmd(adbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "shell [serial] [command...]",
		Short: "Run a shell command on a device",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(*adbPath)
			if err != nil {
				return err
			}
			serial := args[0]
			command := strings.Join(args[1:], " ")
			result, err := client.Shell(cmd.Context(), serial, command)
			if err != nil {
				return err
			}
			fmt.Println(result.Output)
			return nil
		},
	}
}

func screenshotCmd(adbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "screenshot [serial] [output]",
		Short: "Take a screenshot from a device",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(*adbPath)
			if err != nil {
				return err
			}
			serial := args[0]
			output := args[1]
			if err := client.Screenshot(cmd.Context(), serial, output); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Screenshot saved to %s\n", output)
			return nil
		},
	}
}

func installCmd(adbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "install [serial] [apk]",
		Short: "Install an APK on a device",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(*adbPath)
			if err != nil {
				return err
			}
			serial := args[0]
			apkPath := args[1]
			if err := client.InstallAPK(cmd.Context(), serial, apkPath, adb.InstallOptions{}); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Installed %s\n", apkPath)
			return nil
		},
	}
}

func newClient(adbPath string) (*adb.Client, error) {
	if adbPath != "" {
		return adb.NewClientWithPath(adbPath), nil
	}
	cfg, _ := config.Load()
	if cfg.ADBPath != "" {
		return adb.NewClientWithPath(cfg.ADBPath), nil
	}
	return adb.NewClient()
}
