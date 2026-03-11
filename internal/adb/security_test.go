package adb

import (
	"testing"
)

func TestParsePermissionsOutput(t *testing.T) {
	output := `group:android.permission-group.CONTACTS
  permission:android.permission.READ_CONTACTS
  permission:android.permission.WRITE_CONTACTS
  permission:android.permission.GET_ACCOUNTS

group:android.permission-group.CAMERA
  permission:android.permission.CAMERA`

	perms := parsePermissionsOutput(output)
	if len(perms) != 4 {
		t.Fatalf("expected 4 permissions, got %d", len(perms))
	}
	if perms[0] != "android.permission.READ_CONTACTS" {
		t.Fatalf("expected android.permission.READ_CONTACTS, got %s", perms[0])
	}
	if perms[3] != "android.permission.CAMERA" {
		t.Fatalf("expected android.permission.CAMERA, got %s", perms[3])
	}
}

func TestParsePermissionsOutput_Empty(t *testing.T) {
	perms := parsePermissionsOutput("")
	if len(perms) != 0 {
		t.Fatalf("expected 0 permissions, got %d", len(perms))
	}
}

func TestParsePermissionsOutput_NoPermissionLines(t *testing.T) {
	output := "group:android.permission-group.CONTACTS\nlabel:Contacts\ndescription:access contacts\n"
	perms := parsePermissionsOutput(output)
	if len(perms) != 0 {
		t.Fatalf("expected 0 permissions, got %d", len(perms))
	}
}

func TestParseAPKPathOutput(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"package:/data/app/com.example-abc/base.apk\n", "/data/app/com.example-abc/base.apk"},
		{"package:/system/app/Settings.apk", "/system/app/Settings.apk"},
		{"/data/app/test.apk", "/data/app/test.apk"},
		{"", ""},
		{"  package:/data/app/test.apk  \n", "/data/app/test.apk"},
	}
	for _, tc := range cases {
		got := parseAPKPathOutput(tc.input)
		if got != tc.expected {
			t.Fatalf("parseAPKPathOutput(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}
