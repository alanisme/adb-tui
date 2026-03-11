package adb

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) GetClipboard(ctx context.Context, serial string) (string, error) {
	// Parse primary clip text from dumpsys clipboard
	result, err := c.Shell(ctx, serial, "dumpsys clipboard")
	if err != nil {
		return "", fmt.Errorf("get clipboard: %w", err)
	}
	return parseClipboardDump(result.Output), nil
}

func parseClipboardDump(output string) string {
	inClip := false
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Primary clip") || strings.HasPrefix(trimmed, "mPrimaryClip") {
			inClip = true
			continue
		}
		if inClip {
			// Look for text content markers
			if after, ok := strings.CutPrefix(trimmed, "T:"); ok {
				return strings.TrimSpace(after)
			}
			if after, ok := strings.CutPrefix(trimmed, "mText="); ok {
				return strings.TrimSpace(after)
			}
			if strings.Contains(trimmed, "text/plain") {
				continue
			}
			// Plain text content on its own line after the header
			if trimmed != "" && !strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "{") {
				return trimmed
			}
		}
	}
	return ""
}

func (c *Client) SetClipboard(ctx context.Context, serial string, text string) error {
	// Try Clipper app broadcast first (most reliable when installed).
	// We can't reliably detect whether Clipper handled the broadcast from
	// the result code, so always attempt the service call fallback too.
	_, clipperErr := c.ShellArgs(ctx, serial, "am", "broadcast",
		"-a", "clipper.set", "--es", "text", text)

	// Fallback: direct service call (works on many devices without Clipper)
	_, serviceErr := c.Shell(ctx, serial, "service call clipboard 2 i32 1 i32 0 i32 1 s16 'text/plain' s16 "+shellQuote(text)+" i32 0 2>/dev/null")
	if serviceErr == nil || clipperErr == nil {
		return nil
	}
	return fmt.Errorf("set clipboard (may require Clipper app): %w", serviceErr)
}
