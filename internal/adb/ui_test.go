package adb

import "testing"

func TestParseBounds(t *testing.T) {
	tests := []struct {
		input string
		want  Rect
		ok    bool
	}{
		{"[0,0][1080,1920]", Rect{0, 0, 1080, 1920}, true},
		{"[100,200][300,400]", Rect{100, 200, 300, 400}, true},
		{"[0,0][0,0]", Rect{0, 0, 0, 0}, true},
		{"invalid", Rect{}, false},
		{"[0,0]", Rect{}, false},
		{"", Rect{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseBounds(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseBounds(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if got != tt.want {
			t.Errorf("ParseBounds(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRectCenter(t *testing.T) {
	r := Rect{0, 0, 100, 200}
	x, y := r.Center()
	if x != 50 || y != 100 {
		t.Errorf("Center() = (%d, %d), want (50, 100)", x, y)
	}
}

func TestParseUIHierarchy(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<hierarchy rotation="0">
  <node index="0" text="" resource-id="" class="android.widget.FrameLayout" package="com.example" content-desc="" checkable="false" checked="false" clickable="false" enabled="true" focusable="false" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[0,0][1080,1920]">
    <node index="0" text="Hello" resource-id="com.example:id/title" class="android.widget.TextView" package="com.example" content-desc="" checkable="false" checked="false" clickable="true" enabled="true" focusable="true" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[100,200][500,300]"/>
  </node>
</hierarchy>`

	h, err := ParseUIHierarchy(xml)
	if err != nil {
		t.Fatalf("ParseUIHierarchy error: %v", err)
	}
	if h.Rotation != 0 {
		t.Errorf("rotation = %d, want 0", h.Rotation)
	}
	if len(h.Nodes) != 1 {
		t.Fatalf("got %d top nodes, want 1", len(h.Nodes))
	}
	if len(h.Nodes[0].Children) != 1 {
		t.Fatalf("got %d children, want 1", len(h.Nodes[0].Children))
	}
	child := h.Nodes[0].Children[0]
	if child.Text != "Hello" {
		t.Errorf("text = %q, want %q", child.Text, "Hello")
	}
	if child.ResourceID != "com.example:id/title" {
		t.Errorf("resource-id = %q, want %q", child.ResourceID, "com.example:id/title")
	}
	if !child.Clickable {
		t.Error("expected clickable=true")
	}
}

func TestFlattenNodes(t *testing.T) {
	nodes := []UINode{
		{
			Text:    "Parent",
			Bounds:  "[0,0][1080,1920]",
			Enabled: true,
			Children: []UINode{
				{
					Text:      "Child",
					Bounds:    "[100,200][300,400]",
					Clickable: true,
					Enabled:   true,
				},
			},
		},
	}
	elements := FlattenNodes(nodes)
	if len(elements) != 2 {
		t.Fatalf("got %d elements, want 2", len(elements))
	}
	if elements[0].Text != "Parent" {
		t.Errorf("first element text = %q, want %q", elements[0].Text, "Parent")
	}
	if elements[1].Text != "Child" {
		t.Errorf("second element text = %q, want %q", elements[1].Text, "Child")
	}
	if !elements[1].Clickable {
		t.Error("child should be clickable")
	}
}

func TestFindElements(t *testing.T) {
	elements := []UIElement{
		{Text: "Settings", ResourceID: "com.android:id/title", Clickable: true, Enabled: true},
		{Text: "About phone", ResourceID: "com.android:id/summary", Enabled: true},
		{Text: "OK", Class: "android.widget.Button", Clickable: true, Enabled: true},
	}

	// Search by text (case-insensitive)
	matches := FindElements(elements, "settings", "", "")
	if len(matches) != 1 || matches[0].Text != "Settings" {
		t.Errorf("text search: got %d matches", len(matches))
	}

	// Search by resource ID (exact)
	matches = FindElements(elements, "", "com.android:id/summary", "")
	if len(matches) != 1 || matches[0].Text != "About phone" {
		t.Errorf("resourceID search: got %d matches", len(matches))
	}

	// Search by class suffix
	matches = FindElements(elements, "", "", "Button")
	if len(matches) != 1 || matches[0].Text != "OK" {
		t.Errorf("class search: got %d matches", len(matches))
	}

	// No match
	matches = FindElements(elements, "nonexistent", "", "")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestFindElementsByContentDesc(t *testing.T) {
	elements := []UIElement{
		{Text: "", ContentDescription: "Navigate up", Clickable: true, Enabled: true},
	}
	matches := FindElements(elements, "navigate", "", "")
	if len(matches) != 1 {
		t.Errorf("content desc search: got %d matches, want 1", len(matches))
	}
}
