package adb

import (
	"strings"
	"testing"
)

func TestParsePackageDump_Basic(t *testing.T) {
	output := `
    versionCode=33 minSdk=30 targetSdk=33
    versionName=13.0.0
    installerPackageName=com.android.vending
    dataDir=/data/user/0/com.example.app
    userId=10123
    firstInstallTime=2024-01-15 10:00:00
    lastUpdateTime=2024-03-20 08:00:00
    codePath=/data/app/com.example.app-abc123
    pkgFlags=[ HAS_CODE ALLOW_CLEAR_USER_DATA ALLOW_BACKUP ]
    enabled=1
`
	detail := &PackageDetail{
		PackageInfo: PackageInfo{Name: "com.example.app"},
	}
	parsePackageDump(output, detail)

	if detail.VersionCode != "33" {
		t.Fatalf("expected versionCode 33, got %s", detail.VersionCode)
	}
	if detail.VersionName != "13.0.0" {
		t.Fatalf("expected versionName 13.0.0, got %s", detail.VersionName)
	}
	if detail.Installer != "com.android.vending" {
		t.Fatalf("expected installer com.android.vending, got %s", detail.Installer)
	}
	if detail.DataDir != "/data/user/0/com.example.app" {
		t.Fatalf("expected dataDir, got %s", detail.DataDir)
	}
	if detail.UID != "10123" {
		t.Fatalf("expected uid 10123, got %s", detail.UID)
	}
	if detail.APKPath != "/data/app/com.example.app-abc123" {
		t.Fatalf("expected apk path, got %s", detail.APKPath)
	}
	if detail.System {
		t.Fatal("expected non-system package")
	}
}

func TestParsePackageDump_SystemPackage(t *testing.T) {
	output := `
    pkgFlags=[ SYSTEM HAS_CODE ALLOW_CLEAR_USER_DATA ]
`
	detail := &PackageDetail{
		PackageInfo: PackageInfo{Name: "com.android.settings"},
	}
	parsePackageDump(output, detail)

	if !detail.System {
		t.Fatal("expected system package")
	}
}

func TestParsePackageDump_VersionCodeWithExtra(t *testing.T) {
	output := `    versionCode=34 minSdk=31 targetSdk=34`
	detail := &PackageDetail{}
	parsePackageDump(output, detail)
	if detail.VersionCode != "34" {
		t.Fatalf("expected 34, got %s", detail.VersionCode)
	}
}

func TestParsePackageDump_Empty(t *testing.T) {
	detail := &PackageDetail{}
	parsePackageDump("", detail)
	if detail.VersionCode != "" {
		t.Fatal("expected empty versionCode")
	}
}

func TestParsePackageDump_Enabled(t *testing.T) {
	cases := []struct {
		line    string
		enabled bool
	}{
		{"    enabled=0", false},
		{"    enabled=1", true},
		{"    enabled=2", true},
	}
	for _, tc := range cases {
		detail := &PackageDetail{}
		parsePackageDump(tc.line, detail)
		if detail.Enabled != tc.enabled {
			t.Fatalf("for %q: expected enabled=%v, got %v", tc.line, tc.enabled, detail.Enabled)
		}
	}
}

func TestParsePackageDump_FirstAndLastInstall(t *testing.T) {
	output := `
    firstInstallTime=2024-01-01 00:00:00
    lastUpdateTime=2024-06-15 12:30:00
`
	detail := &PackageDetail{}
	parsePackageDump(output, detail)
	if detail.FirstInstall != "2024-01-01 00:00:00" {
		t.Fatalf("expected firstInstallTime, got %s", detail.FirstInstall)
	}
	if detail.LastUpdate != "2024-06-15 12:30:00" {
		t.Fatalf("expected lastUpdateTime, got %s", detail.LastUpdate)
	}
}

func TestInstallOptions(t *testing.T) {
	opts := InstallOptions{
		Reinstall:        true,
		AllowDowngrade:   true,
		GrantPermissions: true,
	}
	if !opts.Reinstall {
		t.Fatal("expected reinstall")
	}
	if !opts.AllowDowngrade {
		t.Fatal("expected allow downgrade")
	}
	if !opts.GrantPermissions {
		t.Fatal("expected grant permissions")
	}
}

func TestListOptions(t *testing.T) {
	opts := ListOptions{
		ShowSystem:     true,
		ShowThirdParty: true,
		ShowDisabled:   true,
		ShowEnabled:    true,
		Filter:         "com.example",
	}
	if !opts.ShowSystem {
		t.Fatal("expected show system")
	}
	if opts.Filter != "com.example" {
		t.Fatalf("expected com.example, got %s", opts.Filter)
	}
}

func TestPackageInfoStruct(t *testing.T) {
	info := PackageInfo{
		Name:        "com.test",
		VersionCode: "1",
		VersionName: "1.0",
	}
	if info.Name != "com.test" {
		t.Fatal("unexpected name")
	}
}

func TestParsePackageDump_VersionNameOnly(t *testing.T) {
	output := `    versionName=2.5.1`
	detail := &PackageDetail{}
	parsePackageDump(output, detail)
	if detail.VersionName != "2.5.1" {
		t.Fatalf("expected 2.5.1, got %s", detail.VersionName)
	}
}

func TestParsePackageList(t *testing.T) {
	output := "package:com.android.settings\npackage:com.android.phone\n"
	pkgs := parsePackageList(output)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "com.android.settings" {
		t.Fatalf("expected com.android.settings, got %s", pkgs[0].Name)
	}
	if pkgs[1].Name != "com.android.phone" {
		t.Fatalf("expected com.android.phone, got %s", pkgs[1].Name)
	}
}

func TestParsePackageList_SecurityException(t *testing.T) {
	output := "java.lang.SecurityException: not allowed to access\n"
	pkgs := parsePackageList(output)
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePackageList_Empty(t *testing.T) {
	pkgs := parsePackageList("")
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestBuildListPackagesArgs(t *testing.T) {
	args := buildListPackagesArgs(ListOptions{}, false)
	if got := strings.Join(args, " "); got != "pm list packages" {
		t.Fatalf("expected basic args, got %s", got)
	}

	args = buildListPackagesArgs(ListOptions{}, true)
	if got := strings.Join(args, " "); got != "pm list packages --user 0" {
		t.Fatalf("expected --user 0 args, got %s", got)
	}

	args = buildListPackagesArgs(ListOptions{ShowSystem: true, ShowThirdParty: true}, false)
	if got := strings.Join(args, " "); got != "pm list packages -s -3" {
		t.Fatalf("expected -s -3 args, got %s", got)
	}

	args = buildListPackagesArgs(ListOptions{Filter: "com.example"}, false)
	if got := strings.Join(args, " "); got != "pm list packages com.example" {
		t.Fatalf("expected filter args, got %s", got)
	}
}
