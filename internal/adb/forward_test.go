package adb

import (
	"testing"
)

func TestParseForwardRules_Multiple(t *testing.T) {
	output := `emulator-5554 tcp:8080 tcp:80
emulator-5554 tcp:9090 tcp:9090
HVA0T18C14000528 tcp:5000 localabstract:app`

	rules := parseForwardRules(output)
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	if rules[0].Serial != "emulator-5554" {
		t.Fatalf("expected emulator-5554, got %s", rules[0].Serial)
	}
	if rules[0].Local != "tcp:8080" {
		t.Fatalf("expected tcp:8080, got %s", rules[0].Local)
	}
	if rules[0].Remote != "tcp:80" {
		t.Fatalf("expected tcp:80, got %s", rules[0].Remote)
	}

	if rules[2].Serial != "HVA0T18C14000528" {
		t.Fatalf("expected HVA0T18C14000528, got %s", rules[2].Serial)
	}
	if rules[2].Remote != "localabstract:app" {
		t.Fatalf("expected localabstract:app, got %s", rules[2].Remote)
	}
}

func TestParseForwardRules_Empty(t *testing.T) {
	rules := parseForwardRules("")
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rules))
	}
}

func TestParseForwardRules_SingleRule(t *testing.T) {
	output := "ABC123 tcp:4000 tcp:4000"
	rules := parseForwardRules(output)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Serial != "ABC123" {
		t.Fatalf("expected ABC123, got %s", rules[0].Serial)
	}
}

func TestParseForwardRules_MalformedLines(t *testing.T) {
	output := `incomplete line
emulator-5554 tcp:8080 tcp:80
another bad`
	rules := parseForwardRules(output)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestParseForwardRules_WhitespaceLines(t *testing.T) {
	output := "\n  \n  emulator-5554 tcp:1234 tcp:5678\n  \n"
	rules := parseForwardRules(output)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestParseReverseRules(t *testing.T) {
	output := "emulator-5554 tcp:8080 tcp:80"
	rules := parseForwardRules(output)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Local != "tcp:8080" {
		t.Fatalf("expected tcp:8080, got %s", rules[0].Local)
	}
	if rules[0].Remote != "tcp:80" {
		t.Fatalf("expected tcp:80, got %s", rules[0].Remote)
	}
}

func TestForwardRuleStruct(t *testing.T) {
	r := ForwardRule{
		Serial: "device1",
		Local:  "tcp:3000",
		Remote: "tcp:3000",
	}
	if r.Serial != "device1" {
		t.Fatal("unexpected serial")
	}
	if r.Local != "tcp:3000" {
		t.Fatal("unexpected local")
	}
}
