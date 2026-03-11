package adb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type PackageInfo struct {
	Name         string
	VersionCode  string
	VersionName  string
	Installer    string
	UID          string
	FirstInstall string
	LastUpdate   string
	Enabled      bool
	System       bool
}

type PackageDetail struct {
	PackageInfo
	Permissions []string
	Activities  []string
	Services    []string
	Receivers   []string
	DataDir     string
	APKPath     string
}

type ListOptions struct {
	ShowSystem     bool
	ShowThirdParty bool
	ShowDisabled   bool
	ShowEnabled    bool
	Filter         string
}

type InstallOptions struct {
	Reinstall        bool
	AllowDowngrade   bool
	GrantPermissions bool
}

type PermissionInfo struct {
	Name    string
	Granted bool
}

func (c *Client) ListPackages(ctx context.Context, serial string, options ListOptions) ([]PackageInfo, error) {
	args := buildListPackagesArgs(options, false)
	result, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		// Some devices fail without --user 0; retry before giving up
		args = buildListPackagesArgs(options, true)
		result2, err2 := c.ShellArgs(ctx, serial, args...)
		if err2 != nil {
			return nil, fmt.Errorf("list packages: %w", err)
		}
		result = result2
	}

	if needsUserFallback(result.Output) {
		args = buildListPackagesArgs(options, true)
		result, err = c.ShellArgs(ctx, serial, args...)
		if err != nil {
			return nil, fmt.Errorf("list packages: %w", err)
		}
	}

	return parsePackageList(result.Output), nil
}

func needsUserFallback(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "securityexception") ||
		strings.Contains(lower, "not allowed to access") ||
		strings.Contains(lower, "not have permission") ||
		strings.Contains(lower, "permission denial")
}

func buildListPackagesArgs(options ListOptions, withUser bool) []string {
	args := []string{"pm", "list", "packages"}
	if withUser {
		args = append(args, "--user", "0")
	}
	if options.ShowSystem {
		args = append(args, "-s")
	}
	if options.ShowThirdParty {
		args = append(args, "-3")
	}
	if options.ShowDisabled {
		args = append(args, "-d")
	}
	if options.ShowEnabled {
		args = append(args, "-e")
	}
	if options.Filter != "" {
		args = append(args, options.Filter)
	}
	return args
}

func parsePackageList(output string) []PackageInfo {
	var packages []PackageInfo
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		name, ok := strings.CutPrefix(line, "package:")
		if !ok {
			continue
		}
		packages = append(packages, PackageInfo{Name: name})
	}
	return packages
}

func (c *Client) InstallAPK(ctx context.Context, serial, path string, options InstallOptions) error {
	args := []string{"install"}
	if options.Reinstall {
		args = append(args, "-r")
	}
	if options.AllowDowngrade {
		args = append(args, "-d")
	}
	if options.GrantPermissions {
		args = append(args, "-g")
	}
	args = append(args, path)

	_, err := c.ExecDevice(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("install apk: %w", err)
	}
	return nil
}

func (c *Client) UninstallPackage(ctx context.Context, serial, pkg string, keepData bool) error {
	args := []string{"uninstall"}
	if keepData {
		args = append(args, "-k")
	}
	args = append(args, pkg)

	_, err := c.ExecDevice(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("uninstall %s: %w", pkg, err)
	}
	return nil
}

func (c *Client) ClearData(ctx context.Context, serial, pkg string) error {
	_, err := c.ShellArgs(ctx, serial, "pm", "clear", pkg)
	if err != nil {
		return fmt.Errorf("clear data %s: %w", pkg, err)
	}
	return nil
}

func (c *Client) ForceStop(ctx context.Context, serial, pkg string) error {
	_, err := c.ShellArgs(ctx, serial, "am", "force-stop", pkg)
	if err != nil {
		return fmt.Errorf("force stop %s: %w", pkg, err)
	}
	return nil
}

func (c *Client) GetPackageInfo(ctx context.Context, serial, pkg string) (*PackageDetail, error) {
	result, err := c.ShellArgs(ctx, serial, "dumpsys", "package", pkg)
	if err != nil {
		return nil, fmt.Errorf("get package info %s: %w", pkg, err)
	}

	detail := &PackageDetail{
		PackageInfo: PackageInfo{Name: pkg},
	}
	parsePackageDump(result.Output, detail)
	return detail, nil
}

func parsePackageDump(output string, detail *PackageDetail) {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		if key, value, ok := strings.Cut(line, "="); ok {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			switch key {
			case "versionCode":
				if v, _, found := strings.Cut(value, " "); found {
					detail.VersionCode = v
				} else {
					detail.VersionCode = value
				}
			case "versionName":
				detail.VersionName = value
			case "installerPackageName":
				detail.Installer = value
			case "dataDir":
				detail.DataDir = value
			case "userId":
				detail.UID = value
			case "firstInstallTime":
				detail.FirstInstall = value
			case "lastUpdateTime":
				detail.LastUpdate = value
			case "enabled":
				detail.Enabled = value != "0"
			}
		}

		if val, ok := strings.CutPrefix(line, "codePath="); ok {
			detail.APKPath = val
		}
		if strings.HasPrefix(line, "pkgFlags=") {
			detail.System = strings.Contains(line, "SYSTEM")
		}
	}
}

func (c *Client) ListAppPermissions(ctx context.Context, serial, pkg string) ([]PermissionInfo, error) {
	result, err := c.ShellArgs(ctx, serial, "dumpsys", "package", pkg)
	if err != nil {
		return nil, fmt.Errorf("list permissions %s: %w", pkg, err)
	}

	requested := parseRequestedPermissions(result.Output)
	granted := parseGrantedPermissions(result.Output)

	perms := make([]PermissionInfo, 0, len(requested))
	for name := range requested {
		perms = append(perms, PermissionInfo{
			Name:    name,
			Granted: granted[name],
		})
	}
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].Name < perms[j].Name
	})
	return perms, nil
}

func parseRequestedPermissions(output string) map[string]bool {
	requested := make(map[string]bool)
	inSection := false
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "requested permissions:" {
			inSection = true
			continue
		}
		if inSection {
			// Section ends at empty line or a new section header (ends with ":")
			if trimmed == "" || strings.HasSuffix(trimmed, ":") {
				break
			}
			// Permission names are dot-separated identifiers, optionally with maxSdkVersion suffix
			name, _, _ := strings.Cut(trimmed, ", maxSdkVersion=")
			if strings.Contains(name, ".") {
				requested[name] = true
			}
		}
	}
	return requested
}

func parseGrantedPermissions(output string) map[string]bool {
	granted := make(map[string]bool)

	type section int
	const (
		sectionNone    section = iota
		sectionRuntime         // "runtime permissions:" — name: granted=true/false
		sectionGranted         // "grantedPermissions:" — one permission per line (older Android)
		sectionInstall         // "install permissions:" — name: granted=true (always granted)
	)
	cur := sectionNone

	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "runtime permissions:":
			cur = sectionRuntime
			continue
		case "grantedPermissions:":
			cur = sectionGranted
			continue
		case "install permissions:":
			cur = sectionInstall
			continue
		}

		if cur == sectionNone {
			continue
		}

		// Empty line or new section header ends current section
		if trimmed == "" || strings.HasSuffix(trimmed, ":") {
			cur = sectionNone
			continue
		}

		switch cur {
		case sectionRuntime:
			if name, rest, ok := strings.Cut(trimmed, ": granted="); ok {
				if rest == "true" {
					granted[name] = true
				}
			}
		case sectionGranted:
			granted[trimmed] = true
		case sectionInstall:
			if name, _, ok := strings.Cut(trimmed, ": granted="); ok {
				granted[name] = true
			}
		}
	}

	return granted
}

func (c *Client) DisablePackage(ctx context.Context, serial, pkg string) error {
	_, err := c.ShellArgs(ctx, serial, "pm", "disable-user", "--user", "0", pkg)
	if err != nil {
		return fmt.Errorf("disable %s: %w", pkg, err)
	}
	return nil
}

func (c *Client) EnablePackage(ctx context.Context, serial, pkg string) error {
	_, err := c.ShellArgs(ctx, serial, "pm", "enable", pkg)
	if err != nil {
		return fmt.Errorf("enable %s: %w", pkg, err)
	}
	return nil
}

func (c *Client) GrantPermission(ctx context.Context, serial, pkg, permission string) error {
	_, err := c.ShellArgs(ctx, serial, "pm", "grant", pkg, permission)
	if err != nil {
		return fmt.Errorf("grant permission: %w", err)
	}
	return nil
}

func (c *Client) RevokePermission(ctx context.Context, serial, pkg, permission string) error {
	_, err := c.ShellArgs(ctx, serial, "pm", "revoke", pkg, permission)
	if err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}
	return nil
}
