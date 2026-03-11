package adb

import (
	"context"
	"fmt"
	"strings"
)

type NotificationInfo struct {
	Key     string
	Package string
	Title   string
	Text    string
	Time    string
}

func (c *Client) ListNotifications(ctx context.Context, serial string) ([]NotificationInfo, error) {
	result, err := c.Shell(ctx, serial, "dumpsys notification --noredact 2>/dev/null || dumpsys notification")
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return parseNotifications(result.Output), nil
}

func parseNotifications(output string) []NotificationInfo {
	var notifications []NotificationInfo
	var current *NotificationInfo
	inExtras := false

	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "NotificationRecord(") {
			if current != nil && current.Package != "" {
				notifications = append(notifications, *current)
			}
			current = &NotificationInfo{}
			if after, ok := strings.CutPrefix(trimmed, "NotificationRecord("); ok {
				current.Key = strings.TrimSuffix(after, ":")
				current.Key = strings.TrimSuffix(current.Key, ")")
			}
			inExtras = false
			continue
		}

		if current == nil {
			continue
		}

		// Detect extras section boundaries
		if strings.HasPrefix(trimmed, "extras=") || strings.HasPrefix(trimmed, "extras:{") || trimmed == "extras:" {
			inExtras = true
			// extras might be inline: extras={android.title=String (text), ...}
			if _, after, ok := strings.Cut(trimmed, "{"); ok {
				parseExtrasInline(after, current)
			}
			continue
		}
		if inExtras {
			if trimmed == "}" || trimmed == "})" {
				inExtras = false
				continue
			}
			// Inside extras section: match android.title=..., android.text=...
			if k, v, ok := strings.Cut(trimmed, "="); ok {
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				switch k {
				case "android.title":
					current.Title = extractNotifValue(v)
				case "android.text":
					current.Text = extractNotifValue(v)
				}
			}
			continue
		}

		// Outside extras: match top-level fields only
		if k, v, ok := strings.Cut(trimmed, "="); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "pkg" {
				current.Package = v
			}
		}
		if after, ok := strings.CutPrefix(trimmed, "postTime="); ok {
			current.Time = after
		}
	}

	if current != nil && current.Package != "" {
		notifications = append(notifications, *current)
	}
	return notifications
}

// parseExtrasInline handles compact extras format: {android.title=String (text), android.text=String (text)}
func parseExtrasInline(s string, n *NotificationInfo) {
	s = strings.TrimSuffix(s, "}")
	for part := range strings.SplitSeq(s, ", android.") {
		part = strings.TrimSpace(part)
		// First part might still have "android." prefix
		part = strings.TrimPrefix(part, "android.")
		if k, v, ok := strings.Cut(part, "="); ok {
			switch k {
			case "title":
				n.Title = extractNotifValue(v)
			case "text":
				n.Text = extractNotifValue(v)
			}
		}
	}
}

func extractNotifValue(s string) string {
	// Format: "String (actual text)" or just the value
	if after, ok := strings.CutPrefix(s, "String ("); ok {
		if len(after) > 0 && after[len(after)-1] == ')' {
			return after[:len(after)-1]
		}
		return after
	}
	return s
}
