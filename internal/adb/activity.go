package adb

import (
	"context"
	"fmt"
	"strings"
)

type Intent struct {
	Action    string
	Data      string
	Type      string
	Component string
	Category  string
	Extras    map[string]string
	Flags     []string
}

// Args returns the intent as a list of arguments safe for ShellArgs.
func (i Intent) Args() []string {
	var parts []string
	if i.Action != "" {
		parts = append(parts, "-a", i.Action)
	}
	if i.Data != "" {
		parts = append(parts, "-d", i.Data)
	}
	if i.Type != "" {
		parts = append(parts, "-t", i.Type)
	}
	if i.Component != "" {
		parts = append(parts, "-n", i.Component)
	}
	if i.Category != "" {
		parts = append(parts, "-c", i.Category)
	}
	for key, value := range i.Extras {
		parts = append(parts, "--es", key, value)
	}
	for _, flag := range i.Flags {
		parts = append(parts, "-f", flag)
	}
	return parts
}

func (c *Client) StartActivity(ctx context.Context, serial string, intent Intent) error {
	args := append([]string{"am", "start"}, intent.Args()...)
	_, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("start activity: %w", err)
	}
	return nil
}

func (c *Client) StartService(ctx context.Context, serial string, intent Intent) error {
	args := append([]string{"am", "startservice"}, intent.Args()...)
	_, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	return nil
}

func (c *Client) SendBroadcast(ctx context.Context, serial string, intent Intent) error {
	args := append([]string{"am", "broadcast"}, intent.Args()...)
	_, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("send broadcast: %w", err)
	}
	return nil
}

func (c *Client) GetCurrentActivity(ctx context.Context, serial string) (string, error) {
	// Match both AOSP "mResumedActivity" and Samsung "ResumedActivity"
	result, err := c.Shell(ctx, serial, "dumpsys activity activities | grep -iE '(mResumed|Resumed)Activity'")
	if err != nil {
		return "", fmt.Errorf("get current activity: %w", err)
	}

	// May return multiple lines; pick the first meaningful one
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: mResumedActivity: ActivityRecord{hash u0 com.pkg/.Activity t123}
		//     or: ResumedActivity: ActivityRecord{hash u0 com.pkg/.Activity t123}
		if _, after, ok := strings.Cut(line, " u0 "); ok {
			if before, _, ok := strings.Cut(after, "}"); ok {
				return strings.TrimSpace(before), nil
			}
			return after, nil
		}
	}
	return strings.TrimSpace(result.Output), nil
}

func (c *Client) ListActivities(ctx context.Context, serial, pkg string) ([]string, error) {
	result, err := c.Shell(ctx, serial, "dumpsys package "+shellQuote(pkg)+" | grep -E 'Activity'")
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}

	var activities []string
	seen := make(map[string]bool)
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		// Look for lines containing package/activity patterns
		if idx := strings.Index(line, pkg+"/"); idx >= 0 {
			rest := line[idx:]
			if end := strings.IndexAny(rest, " }"); end >= 0 {
				rest = rest[:end]
			}
			if !seen[rest] {
				seen[rest] = true
				activities = append(activities, rest)
			}
		}
	}
	return activities, nil
}
