package adb

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

const (
	KeyHome       = 3
	KeyBack       = 4
	KeyCall       = 5
	KeyEndCall    = 6
	KeyDPadUp     = 19
	KeyDPadDown   = 20
	KeyDPadLeft   = 21
	KeyDPadRight  = 22
	KeyDPadCenter = 23
	KeyVolumeUp   = 24
	KeyVolumeDown = 25
	KeyPower      = 26
	KeyCamera     = 27
	KeyClear      = 28
	KeyMenu       = 82
	KeySearch     = 84
	KeyMediaPlay  = 85
	KeyMediaStop  = 86
	KeyMediaNext  = 87
	KeyMediaPrev  = 88
	KeyMediaPause = 127
	KeyMute       = 91
	KeyTab        = 61
	KeyEnter      = 66
	KeyDelete     = 67
	KeyRecents    = 187
	KeyBrightDown = 220
	KeyBrightUp   = 221
	KeySleep      = 223
	KeyWakeUp     = 224
)

func (c *Client) Tap(ctx context.Context, serial string, x, y int) error {
	_, err := c.ShellArgs(ctx, serial, "input", "tap", strconv.Itoa(x), strconv.Itoa(y))
	if err != nil {
		return fmt.Errorf("tap: %w", err)
	}
	return nil
}

func (c *Client) Swipe(ctx context.Context, serial string, x1, y1, x2, y2, durationMs int) error {
	_, err := c.ShellArgs(ctx, serial, "input", "swipe",
		strconv.Itoa(x1), strconv.Itoa(y1),
		strconv.Itoa(x2), strconv.Itoa(y2),
		strconv.Itoa(durationMs))
	if err != nil {
		return fmt.Errorf("swipe: %w", err)
	}
	return nil
}

func (c *Client) KeyEvent(ctx context.Context, serial string, keycode int) error {
	_, err := c.ShellArgs(ctx, serial, "input", "keyevent", strconv.Itoa(keycode))
	if err != nil {
		return fmt.Errorf("key event: %w", err)
	}
	return nil
}

func (c *Client) Text(ctx context.Context, serial, text string) error {
	// adb shell input text requires spaces to be encoded as %s
	escaped := strings.ReplaceAll(text, " ", "%s")
	_, err := c.ShellArgs(ctx, serial, "input", "text", escaped)
	if err != nil {
		return fmt.Errorf("text input: %w", err)
	}
	return nil
}

func (c *Client) LongPress(ctx context.Context, serial string, x, y, durationMs int) error {
	return c.Swipe(ctx, serial, x, y, x, y, durationMs)
}

// GestureParams defines a swipe gesture using fractional screen coordinates (0.0–1.0).
type GestureParams struct {
	StartXFrac, StartYFrac float64
	EndXFrac, EndYFrac     float64
	DurationMs             int
}

// GestureDef is a named gesture preset.
type GestureDef struct {
	Name   string
	Params GestureParams
}

// Gestures is the list of predefined gesture presets.
var Gestures = []GestureDef{
	{"Swipe Up", GestureParams{0.50, 0.75, 0.50, 0.25, 300}},
	{"Swipe Down", GestureParams{0.50, 0.25, 0.50, 0.75, 300}},
	{"Swipe Left", GestureParams{0.75, 0.50, 0.25, 0.50, 300}},
	{"Swipe Right", GestureParams{0.25, 0.50, 0.75, 0.50, 300}},
	{"Scroll Up", GestureParams{0.50, 0.60, 0.50, 0.40, 200}},
	{"Scroll Down", GestureParams{0.50, 0.40, 0.50, 0.60, 200}},
	{"Fling Up", GestureParams{0.50, 0.80, 0.50, 0.20, 80}},
	{"Fling Down", GestureParams{0.50, 0.20, 0.50, 0.80, 80}},
	{"Pull Down", GestureParams{0.50, 0.02, 0.50, 0.60, 300}},
	{"Pull Up", GestureParams{0.50, 0.98, 0.50, 0.40, 300}},
}

// HumanSwipe performs a swipe gesture with small random perturbations to
// simulate natural human touch behavior. Adds ±10px endpoint jitter and
// ±5px perpendicular drift so the path is never a perfect straight line.
func (c *Client) HumanSwipe(ctx context.Context, serial string, screenW, screenH int, p GestureParams) error {
	// Convert fractional coordinates to pixels
	x1 := int(p.StartXFrac * float64(screenW))
	y1 := int(p.StartYFrac * float64(screenH))
	x2 := int(p.EndXFrac * float64(screenW))
	y2 := int(p.EndYFrac * float64(screenH))

	// Determine primary direction BEFORE adding jitter
	dx := x2 - x1
	dy := y2 - y1

	// Add endpoint jitter (±10px)
	x1 += rand.IntN(21) - 10
	y1 += rand.IntN(21) - 10
	x2 += rand.IntN(21) - 10
	y2 += rand.IntN(21) - 10

	// Add perpendicular drift so the line isn't perfectly straight.
	// For a mostly-vertical swipe, drift horizontally; for horizontal, drift vertically.
	drift := rand.IntN(11) - 5
	if max(dy, -dy) > max(dx, -dx) {
		x1 += drift
		x2 += drift / 2
	} else {
		y1 += drift
		y2 += drift / 2
	}

	// Add slight duration variance (±15%)
	dur := p.DurationMs
	if variance := p.DurationMs * 15 / 100; variance > 0 {
		dur += rand.IntN(2*variance+1) - variance
	}
	dur = max(dur, 20)

	// Clamp to screen bounds
	x1 = clampInt(x1, 0, screenW-1)
	y1 = clampInt(y1, 0, screenH-1)
	x2 = clampInt(x2, 0, screenW-1)
	y2 = clampInt(y2, 0, screenH-1)

	return c.Swipe(ctx, serial, x1, y1, x2, y2, dur)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
