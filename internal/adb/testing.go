package adb

import (
	"context"
	"fmt"
)

type MonkeyOptions struct {
	Seed                     int64
	Throttle                 int
	IgnoreCrashes            bool
	IgnoreTimeouts           bool
	IgnoreSecurityExceptions bool
}

type InstrumentOptions struct {
	Runner            string
	Arguments         map[string]string
	RawOutput         bool
	NoWindowAnimation bool
}

func (c *Client) RunMonkey(ctx context.Context, serial, pkg string, events int, options MonkeyOptions) (*ExecResult, error) {
	args := []string{"monkey", "-p", pkg}
	if options.Seed != 0 {
		args = append(args, "-s", fmt.Sprintf("%d", options.Seed))
	}
	if options.Throttle > 0 {
		args = append(args, "--throttle", fmt.Sprintf("%d", options.Throttle))
	}
	if options.IgnoreCrashes {
		args = append(args, "--ignore-crashes")
	}
	if options.IgnoreTimeouts {
		args = append(args, "--ignore-timeouts")
	}
	if options.IgnoreSecurityExceptions {
		args = append(args, "--ignore-security-exceptions")
	}
	args = append(args, "-v", fmt.Sprintf("%d", events))

	result, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return result, fmt.Errorf("run monkey: %w", err)
	}
	return result, nil
}

func (c *Client) RunInstrumentation(ctx context.Context, serial, component string, options InstrumentOptions) (*ExecResult, error) {
	args := []string{"am", "instrument"}
	if options.RawOutput {
		args = append(args, "-r")
	}
	if options.NoWindowAnimation {
		args = append(args, "--no-window-animation")
	}
	if options.Runner != "" {
		args = append(args, "-e", "class", options.Runner)
	}
	for k, v := range options.Arguments {
		args = append(args, "-e", k, v)
	}
	args = append(args, "-w", component)

	result, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return result, fmt.Errorf("run instrumentation: %w", err)
	}
	return result, nil
}

