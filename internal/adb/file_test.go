package adb

import (
	"testing"
)

func TestParseLsLine_RegularFile(t *testing.T) {
	line := "-rw-rw-r-- 1 root sdcard_rw 12345 2024-01-15 10:30 test.txt"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if fi.Name != "test.txt" {
		t.Fatalf("expected test.txt, got %s", fi.Name)
	}
	if fi.IsDir {
		t.Fatal("expected not dir")
	}
	if fi.IsLink {
		t.Fatal("expected not link")
	}
	if fi.Permissions != "-rw-rw-r--" {
		t.Fatalf("expected -rw-rw-r--, got %s", fi.Permissions)
	}
	if fi.Size != 12345 {
		t.Fatalf("expected 12345, got %d", fi.Size)
	}
	if fi.ModTime.IsZero() {
		t.Fatal("expected non-zero mod time")
	}
	if fi.ModTime.Month() != 1 || fi.ModTime.Day() != 15 {
		t.Fatalf("expected Jan 15, got %v", fi.ModTime)
	}
}

func TestParseLsLine_Directory(t *testing.T) {
	line := "drwxrwx--x 2 root sdcard_rw 4096 2024-03-20 08:00 Downloads"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if !fi.IsDir {
		t.Fatal("expected dir")
	}
	if fi.Name != "Downloads" {
		t.Fatalf("expected Downloads, got %s", fi.Name)
	}
	if fi.Permissions != "drwxrwx--x" {
		t.Fatalf("expected drwxrwx--x, got %s", fi.Permissions)
	}
}

func TestParseLsLine_Symlink(t *testing.T) {
	line := "lrwxrwxrwx 1 root root 10 2024-02-01 12:00 link -> /data/target"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if !fi.IsLink {
		t.Fatal("expected link")
	}
	if fi.Name != "link" {
		t.Fatalf("expected link, got %s", fi.Name)
	}
	if fi.LinkTarget != "/data/target" {
		t.Fatalf("expected /data/target, got %s", fi.LinkTarget)
	}
}

func TestParseLsLine_SymlinkWithSpaces(t *testing.T) {
	line := "lrwxrwxrwx 1 root root 20 2024-02-01 12:00 my link -> /data/my target"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if fi.Name != "my link" {
		t.Fatalf("expected 'my link', got %s", fi.Name)
	}
	if fi.LinkTarget != "/data/my target" {
		t.Fatalf("expected '/data/my target', got %s", fi.LinkTarget)
	}
}

func TestParseLsLine_TooFewFields(t *testing.T) {
	_, ok := parseLsLine("-rw-r-- 1 root")
	if ok {
		t.Fatal("expected not ok for too few fields")
	}
}

func TestParseLsLine_Empty(t *testing.T) {
	_, ok := parseLsLine("")
	if ok {
		t.Fatal("expected not ok for empty")
	}
}

func TestParseLsLine_FileWithSpacesInName(t *testing.T) {
	line := "-rw-rw-r-- 1 root sdcard_rw 100 2024-06-01 09:30 my file name.txt"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if fi.Name != "my file name.txt" {
		t.Fatalf("expected 'my file name.txt', got %s", fi.Name)
	}
}

func TestParseLsLine_ZeroSize(t *testing.T) {
	line := "-rw-r--r-- 1 root root 0 2024-01-01 00:00 empty.txt"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if fi.Size != 0 {
		t.Fatalf("expected size 0, got %d", fi.Size)
	}
}

func TestParseLsLine_LargeSize(t *testing.T) {
	line := "-rw-rw-r-- 1 root sdcard_rw 2147483648 2024-05-01 12:00 big.bin"
	fi, ok := parseLsLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if fi.Size != 2147483648 {
		t.Fatalf("expected 2147483648, got %d", fi.Size)
	}
}

func TestParseLsLine_Permissions(t *testing.T) {
	cases := []struct {
		perms string
		isDir bool
		isLnk bool
	}{
		{"-rwxrwxrwx", false, false},
		{"drwxr-xr-x", true, false},
		{"lrwxrwxrwx", false, true},
	}
	for _, tc := range cases {
		line := tc.perms + " 1 root root 100 2024-01-01 00:00 name"
		fi, ok := parseLsLine(line)
		if !ok {
			t.Fatalf("expected ok for perms %s", tc.perms)
		}
		if fi.IsDir != tc.isDir {
			t.Fatalf("perms %s: expected isDir=%v", tc.perms, tc.isDir)
		}
		if fi.IsLink != tc.isLnk {
			t.Fatalf("perms %s: expected isLink=%v", tc.perms, tc.isLnk)
		}
	}
}

func TestFileInfoStruct(t *testing.T) {
	fi := FileInfo{
		Name:        "test",
		Size:        42,
		Permissions: "-rw-r--r--",
		IsDir:       false,
		IsLink:      false,
	}
	if fi.Name != "test" {
		t.Fatal("unexpected name")
	}
}
