package adb

import "testing"

func TestNeedsUserFallback(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"security exception", "java.lang.SecurityException: blah", true},
		{"not allowed", "not allowed to access packages", true},
		{"no permission", "does not have permission to read", true},
		{"permission denial", "Permission Denial: reading", true},
		{"normal output", "package:com.android.settings\npackage:com.android.phone", false},
		{"empty", "", false},
		{"case insensitive", "SECURITYEXCEPTION occurred", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsUserFallback(tt.output)
			if got != tt.want {
				t.Errorf("needsUserFallback(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestParseRequestedPermissions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]bool
	}{
		{
			name: "standard permissions",
			input: `  Packages:
    requested permissions:
      android.permission.INTERNET
      android.permission.ACCESS_NETWORK_STATE
      android.permission.CAMERA, maxSdkVersion=30
    install permissions:
      android.permission.INTERNET: granted=true`,
			want: map[string]bool{
				"android.permission.INTERNET":             true,
				"android.permission.ACCESS_NETWORK_STATE": true,
				"android.permission.CAMERA":               true,
			},
		},
		{
			name:  "empty output",
			input: "",
			want:  map[string]bool{},
		},
		{
			name: "no requested section",
			input: `  install permissions:
      android.permission.INTERNET: granted=true`,
			want: map[string]bool{},
		},
		{
			name: "empty section",
			input: `    requested permissions:

    install permissions:`,
			want: map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRequestedPermissions(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d permissions, want %d", len(got), len(tt.want))
			}
			for k := range tt.want {
				if !got[k] {
					t.Errorf("missing permission %q", k)
				}
			}
		})
	}
}

func TestParseGrantedPermissions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]bool
	}{
		{
			name: "runtime permissions",
			input: `    runtime permissions:
      android.permission.CAMERA: granted=true
      android.permission.RECORD_AUDIO: granted=false
      android.permission.READ_CONTACTS: granted=true
    other section:`,
			want: map[string]bool{
				"android.permission.CAMERA":        true,
				"android.permission.READ_CONTACTS": true,
			},
		},
		{
			name: "install permissions",
			input: `    install permissions:
      android.permission.INTERNET: granted=true
      android.permission.ACCESS_NETWORK_STATE: granted=true`,
			want: map[string]bool{
				"android.permission.INTERNET":             true,
				"android.permission.ACCESS_NETWORK_STATE": true,
			},
		},
		{
			name: "granted permissions (older Android)",
			input: `    grantedPermissions:
      android.permission.INTERNET
      android.permission.CAMERA`,
			want: map[string]bool{
				"android.permission.INTERNET": true,
				"android.permission.CAMERA":   true,
			},
		},
		{
			name: "mixed sections",
			input: `    runtime permissions:
      android.permission.CAMERA: granted=true
    install permissions:
      android.permission.INTERNET: granted=true
    grantedPermissions:
      android.permission.VIBRATE`,
			want: map[string]bool{
				"android.permission.CAMERA":   true,
				"android.permission.INTERNET": true,
				"android.permission.VIBRATE":  true,
			},
		},
		{
			name:  "empty",
			input: "",
			want:  map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGrantedPermissions(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d granted, want %d\ngot: %v", len(got), len(tt.want), got)
			}
			for k := range tt.want {
				if !got[k] {
					t.Errorf("missing granted permission %q", k)
				}
			}
		})
	}
}
