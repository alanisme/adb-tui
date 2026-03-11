package adb

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// UINode represents a single node in the Android UI hierarchy.
type UINode struct {
	Index              int       `xml:"index,attr"`
	Text               string    `xml:"text,attr"`
	ResourceID         string    `xml:"resource-id,attr"`
	Class              string    `xml:"class,attr"`
	Package            string    `xml:"package,attr"`
	ContentDescription string    `xml:"content-desc,attr"`
	Checkable          bool      `xml:"checkable,attr"`
	Checked            bool      `xml:"checked,attr"`
	Clickable          bool      `xml:"clickable,attr"`
	Enabled            bool      `xml:"enabled,attr"`
	Focusable          bool      `xml:"focusable,attr"`
	Focused            bool      `xml:"focused,attr"`
	Scrollable         bool      `xml:"scrollable,attr"`
	LongClickable      bool      `xml:"long-clickable,attr"`
	Password           bool      `xml:"password,attr"`
	Selected           bool      `xml:"selected,attr"`
	Bounds             string    `xml:"bounds,attr"`
	Children           []UINode  `xml:"node"`
}

// UIHierarchy is the root of a uiautomator dump.
type UIHierarchy struct {
	XMLName  xml.Name `xml:"hierarchy"`
	Rotation int      `xml:"rotation,attr"`
	Nodes    []UINode `xml:"node"`
}

// Rect holds parsed bounds.
type Rect struct {
	Left, Top, Right, Bottom int
}

// Center returns the center point of the rectangle.
func (r Rect) Center() (int, int) {
	return (r.Left + r.Right) / 2, (r.Top + r.Bottom) / 2
}

// ParseBounds parses a bounds string like "[0,0][1080,1920]".
func ParseBounds(s string) (Rect, bool) {
	// Format: [left,top][right,bottom]
	s = strings.ReplaceAll(s, "][", ",")
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return Rect{}, false
	}
	nums := make([]int, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Rect{}, false
		}
		nums[i] = n
	}
	return Rect{nums[0], nums[1], nums[2], nums[3]}, true
}

// DumpUIHierarchy captures the current UI hierarchy as XML.
func (c *Client) DumpUIHierarchy(ctx context.Context, serial string) (string, error) {
	remotePath := "/sdcard/window_dump.xml"
	_, err := c.ShellArgs(ctx, serial, "uiautomator", "dump", remotePath)
	if err != nil {
		return "", fmt.Errorf("uiautomator dump: %w", err)
	}
	content, err := c.Cat(ctx, serial, remotePath)
	if err != nil {
		return "", fmt.Errorf("read ui dump: %w", err)
	}
	_, _ = c.ShellArgs(ctx, serial, "rm", remotePath)
	return content, nil
}

// ParseUIHierarchy parses XML from uiautomator dump into a structured tree.
func ParseUIHierarchy(xmlData string) (*UIHierarchy, error) {
	var h UIHierarchy
	if err := xml.Unmarshal([]byte(xmlData), &h); err != nil {
		return nil, fmt.Errorf("parse ui hierarchy: %w", err)
	}
	return &h, nil
}

// UIElement is a flattened representation of a UI node with its bounds.
type UIElement struct {
	Text               string `json:"text,omitempty"`
	ResourceID         string `json:"resource_id,omitempty"`
	Class              string `json:"class,omitempty"`
	Package            string `json:"package,omitempty"`
	ContentDescription string `json:"content_desc,omitempty"`
	Clickable          bool   `json:"clickable"`
	Enabled            bool   `json:"enabled"`
	Checked            bool   `json:"checked"`
	Focused            bool   `json:"focused"`
	Selected           bool   `json:"selected"`
	Scrollable         bool   `json:"scrollable"`
	Bounds             Rect   `json:"bounds"`
}

// FlattenNodes walks the UI tree and returns all nodes as a flat list.
func FlattenNodes(nodes []UINode) []UIElement {
	var result []UIElement
	var walk func([]UINode)
	walk = func(ns []UINode) {
		for _, n := range ns {
			bounds, _ := ParseBounds(n.Bounds)
			result = append(result, UIElement{
				Text:               n.Text,
				ResourceID:         n.ResourceID,
				Class:              n.Class,
				Package:            n.Package,
				ContentDescription: n.ContentDescription,
				Clickable:          n.Clickable,
				Enabled:            n.Enabled,
				Checked:            n.Checked,
				Focused:            n.Focused,
				Selected:           n.Selected,
				Scrollable:         n.Scrollable,
				Bounds:             bounds,
			})
			walk(n.Children)
		}
	}
	walk(nodes)
	return result
}

// FindElements searches the UI hierarchy for elements matching the given criteria.
// Matching is case-insensitive substring for text and content_desc, exact match for resource_id.
// Results are sorted by position (top-to-bottom, then left-to-right) for stable indexing.
func FindElements(elements []UIElement, text, resourceID, className string) []UIElement {
	textLower := strings.ToLower(text)
	var matches []UIElement
	for _, e := range elements {
		if resourceID != "" && e.ResourceID == resourceID {
			matches = append(matches, e)
			continue
		}
		if text != "" {
			if strings.Contains(strings.ToLower(e.Text), textLower) ||
				strings.Contains(strings.ToLower(e.ContentDescription), textLower) {
				matches = append(matches, e)
				continue
			}
		}
		if className != "" && strings.HasSuffix(e.Class, className) {
			matches = append(matches, e)
		}
	}
	// Sort by position: top-to-bottom, then left-to-right for stable indexing
	slices.SortFunc(matches, func(a, b UIElement) int {
		if c := cmp.Compare(a.Bounds.Top, b.Bounds.Top); c != 0 {
			return c
		}
		return cmp.Compare(a.Bounds.Left, b.Bounds.Left)
	})
	return matches
}

// FindAndTapElement dumps the UI, finds an element by text or resource ID, and taps it.
// The index parameter selects which match to tap when multiple elements match (0-based).
func (c *Client) FindAndTapElement(ctx context.Context, serial, text, resourceID string, index int) error {
	xmlData, err := c.DumpUIHierarchy(ctx, serial)
	if err != nil {
		return err
	}
	h, err := ParseUIHierarchy(xmlData)
	if err != nil {
		return err
	}
	elements := FlattenNodes(h.Nodes)
	matches := FindElements(elements, text, resourceID, "")
	if len(matches) == 0 {
		return fmt.Errorf("element not found: text=%q resource_id=%q", text, resourceID)
	}
	if index < 0 || index >= len(matches) {
		return fmt.Errorf("index %d out of range, found %d matches", index, len(matches))
	}
	x, y := matches[index].Bounds.Center()
	return c.Tap(ctx, serial, x, y)
}

// WaitForElement polls the UI hierarchy until an element matching text or resourceID appears.
func (c *Client) WaitForElement(ctx context.Context, serial, text, resourceID string, timeoutMs int) (*UIElement, error) {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		xmlData, err := c.DumpUIHierarchy(ctx, serial)
		if err != nil {
			time.Sleep(interval)
			continue
		}
		h, err := ParseUIHierarchy(xmlData)
		if err != nil {
			time.Sleep(interval)
			continue
		}
		elements := FlattenNodes(h.Nodes)
		matches := FindElements(elements, text, resourceID, "")
		if len(matches) > 0 {
			return &matches[0], nil
		}
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("timeout waiting for element: text=%q resource_id=%q", text, resourceID)
}

// GetFocusedApp returns the package name of the currently focused application.
func (c *Client) GetFocusedApp(ctx context.Context, serial string) (string, error) {
	result, err := c.Shell(ctx, serial, "dumpsys window displays | grep mCurrentFocus")
	if err != nil {
		return "", fmt.Errorf("get focused app: %w", err)
	}
	// Format: mCurrentFocus=Window{hash u0 com.package/com.package.Activity}
	output := strings.TrimSpace(result.Output)
	if _, after, ok := strings.Cut(output, " u0 "); ok {
		if before, _, found := strings.Cut(after, "/"); found {
			return before, nil
		}
		pkg := strings.TrimSuffix(after, "}")
		return pkg, nil
	}
	return output, nil
}

// IsScreenOn checks whether the device screen is currently on.
func (c *Client) IsScreenOn(ctx context.Context, serial string) (bool, error) {
	result, err := c.Shell(ctx, serial, "dumpsys power | grep 'Display Power'")
	if err != nil {
		return false, fmt.Errorf("check screen: %w", err)
	}
	return strings.Contains(result.Output, "state=ON"), nil
}

// ScreenOn wakes the device screen.
func (c *Client) ScreenOn(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyWakeUp)
}

// ScreenOff turns off the device screen.
func (c *Client) ScreenOff(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeySleep)
}

// WriteFile writes text content to a file on the device.
func (c *Client) WriteFile(ctx context.Context, serial, path, content string) error {
	// Use base64 encoding to safely transfer arbitrary content without shell escaping issues.
	// This avoids problems with special characters, newlines, quotes, etc.
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	_, err := c.Shell(ctx, serial, "echo "+shellQuote(encoded)+" | base64 -d > "+shellQuote(path))
	if err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

// OpenURL opens a URL in the default browser.
func (c *Client) OpenURL(ctx context.Context, serial, url string) error {
	return c.StartActivity(ctx, serial, Intent{
		Action: "android.intent.action.VIEW",
		Data:   url,
	})
}
