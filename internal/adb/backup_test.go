package adb

import (
	"slices"
	"testing"
)

func TestBuildBackupArgs_Defaults(t *testing.T) {
	args := buildBackupArgs("/tmp/backup.ab", BackupOptions{})
	expected := []string{"backup", "-f", "/tmp/backup.ab", "-noapk", "-noobb", "-noshared", "-nosystem"}
	if !slices.Equal(args, expected) {
		t.Fatalf("expected %v, got %v", expected, args)
	}
}

func TestBuildBackupArgs_AllEnabled(t *testing.T) {
	opts := BackupOptions{
		APK:    true,
		OBB:    true,
		Shared: true,
		All:    true,
		System: true,
	}
	args := buildBackupArgs("/tmp/backup.ab", opts)

	if !slices.Contains(args, "-apk") {
		t.Fatalf("missing -apk: %v", args)
	}
	if !slices.Contains(args, "-obb") {
		t.Fatalf("missing -obb: %v", args)
	}
	if !slices.Contains(args, "-shared") {
		t.Fatalf("missing -shared: %v", args)
	}
	if !slices.Contains(args, "-all") {
		t.Fatalf("missing -all: %v", args)
	}
	if !slices.Contains(args, "-system") {
		t.Fatalf("missing -system: %v", args)
	}
	if slices.Contains(args, "-noapk") {
		t.Fatalf("unexpected -noapk: %v", args)
	}
}

func TestBuildBackupArgs_WithPackages(t *testing.T) {
	opts := BackupOptions{
		Packages: []string{"com.example.app1", "com.example.app2"},
	}
	args := buildBackupArgs("/tmp/out.ab", opts)

	last2 := args[len(args)-2:]
	if last2[0] != "com.example.app1" || last2[1] != "com.example.app2" {
		t.Fatalf("expected packages at end, got %v", args)
	}
}

func TestBuildBackupArgs_OutputPath(t *testing.T) {
	args := buildBackupArgs("/sdcard/my-backup.ab", BackupOptions{})
	if args[0] != "backup" {
		t.Fatalf("expected backup command, got %s", args[0])
	}
	if args[1] != "-f" {
		t.Fatalf("expected -f flag, got %s", args[1])
	}
	if args[2] != "/sdcard/my-backup.ab" {
		t.Fatalf("expected output path, got %s", args[2])
	}
}

func TestBuildBackupArgs_MixedOptions(t *testing.T) {
	opts := BackupOptions{
		APK:    true,
		OBB:    false,
		Shared: true,
		All:    true,
		System: false,
	}
	args := buildBackupArgs("/out.ab", opts)
	if !slices.Contains(args, "-apk") {
		t.Fatalf("missing -apk: %v", args)
	}
	if !slices.Contains(args, "-noobb") {
		t.Fatalf("missing -noobb: %v", args)
	}
	if !slices.Contains(args, "-shared") {
		t.Fatalf("missing -shared: %v", args)
	}
	if !slices.Contains(args, "-nosystem") {
		t.Fatalf("missing -nosystem: %v", args)
	}
}
