package adb

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type LogLevel string

const (
	LogVerbose LogLevel = "V"
	LogDebug   LogLevel = "D"
	LogInfo    LogLevel = "I"
	LogWarn    LogLevel = "W"
	LogError   LogLevel = "E"
	LogFatal   LogLevel = "F"
)

type LogEntry struct {
	Timestamp time.Time
	PID       string
	TID       string
	Level     LogLevel
	Tag       string
	Message   string
}

type LogcatOptions struct {
	Filter string
	Format string
	Buffer string
	Since  string
	Count  int
}

func (c *Client) LogcatStream(ctx context.Context, serial string, options LogcatOptions) (<-chan LogEntry, error) {
	args := c.buildLogcatArgs(serial, options, false)

	cmd := exec.CommandContext(ctx, c.adbPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("logcat stream: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("logcat stream: %w", err)
	}

	ch := make(chan LogEntry, 256)
	go func() {
		defer close(ch)
		defer cmd.Wait()
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			entry, ok := parseLogLine(scanner.Text())
			if !ok {
				continue
			}
			select {
			case ch <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (c *Client) LogcatClear(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "logcat", "-c")
	if err != nil {
		return fmt.Errorf("logcat clear: %w", err)
	}
	return nil
}

func (c *Client) LogcatDump(ctx context.Context, serial string, options LogcatOptions) ([]LogEntry, error) {
	args := c.buildLogcatArgs(serial, options, true)

	result, err := c.Exec(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("logcat dump: %w", err)
	}

	var entries []LogEntry
	for line := range strings.SplitSeq(result.Output, "\n") {
		entry, ok := parseLogLine(line)
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (c *Client) buildLogcatArgs(serial string, options LogcatOptions, dump bool) []string {
	var args []string
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "logcat")

	if dump {
		args = append(args, "-d")
	}
	if options.Format != "" {
		args = append(args, "-v", options.Format)
	}
	if options.Buffer != "" {
		args = append(args, "-b", options.Buffer)
	}
	if options.Since != "" {
		args = append(args, "-T", options.Since)
	}
	if options.Count > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", options.Count))
	}
	if options.Filter != "" {
		args = append(args, options.Filter)
	}
	return args
}

// parseLogLine parses a threadtime-formatted logcat line.
// Format: MM-DD HH:MM:SS.mmm  PID  TID LEVEL TAG: MESSAGE
func parseLogLine(line string) (LogEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "-----") {
		return LogEntry{}, false
	}

	// Minimal parse: find level character and split around it.
	// Expected: "01-02 03:04:05.678  1234  5678 I Tag: message"
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return LogEntry{}, false
	}

	dateStr := fields[0] + " " + fields[1]
	ts, _ := time.Parse("01-02 15:04:05.000", dateStr)
	if ts.Year() == 0 {
		ts = ts.AddDate(time.Now().Year(), 0, 0)
	}

	pid := fields[2]
	tid := fields[3]
	level := LogLevel(fields[4])

	tag := strings.TrimSuffix(fields[5], ":")
	var message string
	if len(fields) > 6 {
		message = strings.Join(fields[6:], " ")
	}

	return LogEntry{
		Timestamp: ts,
		PID:       pid,
		TID:       tid,
		Level:     level,
		Tag:       tag,
		Message:   message,
	}, true
}
