package adb

import (
	"context"
	"fmt"
)

type BackupOptions struct {
	APK      bool
	OBB      bool
	Shared   bool
	All      bool
	System   bool
	Packages []string
}

func (c *Client) Bugreport(ctx context.Context, serial, outputPath string) error {
	_, err := c.ExecDevice(ctx, serial, "bugreport", outputPath)
	if err != nil {
		return fmt.Errorf("bugreport: %w", err)
	}
	return nil
}

func (c *Client) Sideload(ctx context.Context, serial, otaPath string) error {
	_, err := c.ExecDevice(ctx, serial, "sideload", otaPath)
	if err != nil {
		return fmt.Errorf("sideload: %w", err)
	}
	return nil
}

func buildBackupArgs(outputPath string, options BackupOptions) []string {
	args := []string{"backup", "-f", outputPath}
	if options.APK {
		args = append(args, "-apk")
	} else {
		args = append(args, "-noapk")
	}
	if options.OBB {
		args = append(args, "-obb")
	} else {
		args = append(args, "-noobb")
	}
	if options.Shared {
		args = append(args, "-shared")
	} else {
		args = append(args, "-noshared")
	}
	if options.All {
		args = append(args, "-all")
	}
	if options.System {
		args = append(args, "-system")
	} else {
		args = append(args, "-nosystem")
	}
	args = append(args, options.Packages...)
	return args
}

func (c *Client) Backup(ctx context.Context, serial, outputPath string, options BackupOptions) error {
	args := buildBackupArgs(outputPath, options)
	_, err := c.ExecDevice(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	return nil
}

func (c *Client) Restore(ctx context.Context, serial, backupPath string) error {
	_, err := c.ExecDevice(ctx, serial, "restore", backupPath)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	return nil
}
