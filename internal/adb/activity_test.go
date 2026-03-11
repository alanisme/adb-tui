package adb

import (
	"testing"
)

func TestIntentArgs_AllFields(t *testing.T) {
	intent := Intent{
		Action:    "android.intent.action.VIEW",
		Data:      "https://example.com",
		Type:      "text/html",
		Component: "com.example/.MainActivity",
		Category:  "android.intent.category.DEFAULT",
		Flags:     []string{"0x10000000"},
	}
	args := intent.Args()

	expected := map[string]string{
		"-a": "android.intent.action.VIEW",
		"-d": "https://example.com",
		"-t": "text/html",
		"-n": "com.example/.MainActivity",
		"-c": "android.intent.category.DEFAULT",
		"-f": "0x10000000",
	}

	for i := 0; i < len(args)-1; i += 2 {
		flag := args[i]
		val := args[i+1]
		if exp, ok := expected[flag]; ok {
			if val != exp {
				t.Fatalf("flag %s: expected %s, got %s", flag, exp, val)
			}
			delete(expected, flag)
		}
	}
	if len(expected) > 0 {
		t.Fatalf("missing flags: %v", expected)
	}
}

func TestIntentArgs_Empty(t *testing.T) {
	intent := Intent{}
	args := intent.Args()
	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d: %v", len(args), args)
	}
}

func TestIntentArgs_ActionOnly(t *testing.T) {
	intent := Intent{Action: "android.intent.action.MAIN"}
	args := intent.Args()
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "-a" || args[1] != "android.intent.action.MAIN" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestIntentArgs_ComponentOnly(t *testing.T) {
	intent := Intent{Component: "com.example/.Activity"}
	args := intent.Args()
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "-n" || args[1] != "com.example/.Activity" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestIntentArgs_DataAndType(t *testing.T) {
	intent := Intent{
		Data: "content://contacts/1",
		Type: "vnd.android.cursor.item/contact",
	}
	args := intent.Args()
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestIntentArgs_WithExtras(t *testing.T) {
	intent := Intent{
		Action: "com.example.ACTION",
		Extras: map[string]string{
			"key1": "value1",
		},
	}
	args := intent.Args()

	foundExtra := false
	for i := range len(args) - 2 {
		if args[i] == "--es" && args[i+1] == "key1" && args[i+2] == "value1" {
			foundExtra = true
			break
		}
	}
	if !foundExtra {
		t.Fatalf("expected --es key1 value1 in args: %v", args)
	}
}

func TestIntentArgs_MultipleFlags(t *testing.T) {
	intent := Intent{
		Flags: []string{"0x10000000", "0x20000000"},
	}
	args := intent.Args()
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}

	flagCount := 0
	for _, a := range args {
		if a == "-f" {
			flagCount++
		}
	}
	if flagCount != 2 {
		t.Fatalf("expected 2 -f flags, got %d", flagCount)
	}
}

func TestIntentStruct(t *testing.T) {
	i := Intent{
		Action:    "test",
		Data:      "data",
		Type:      "type",
		Component: "comp",
		Category:  "cat",
	}
	if i.Action != "test" {
		t.Fatal("unexpected action")
	}
}
