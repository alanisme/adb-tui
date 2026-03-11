package adb

import (
	"context"
	"fmt"
	"strings"
)

type ForwardRule struct {
	Serial string
	Local  string
	Remote string
}

func (c *Client) Forward(ctx context.Context, serial, local, remote string) error {
	_, err := c.ExecDevice(ctx, serial, "forward", local, remote)
	if err != nil {
		return fmt.Errorf("forward: %w", err)
	}
	return nil
}

func (c *Client) ForwardList(ctx context.Context, serial string) ([]ForwardRule, error) {
	result, err := c.ExecDevice(ctx, serial, "forward", "--list")
	if err != nil {
		return nil, fmt.Errorf("forward list: %w", err)
	}
	return parseForwardRules(result.Output), nil
}

func (c *Client) ForwardRemove(ctx context.Context, serial, local string) error {
	_, err := c.ExecDevice(ctx, serial, "forward", "--remove", local)
	if err != nil {
		return fmt.Errorf("forward remove: %w", err)
	}
	return nil
}

func (c *Client) ForwardRemoveAll(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "forward", "--remove-all")
	if err != nil {
		return fmt.Errorf("forward remove all: %w", err)
	}
	return nil
}

func (c *Client) Reverse(ctx context.Context, serial, remote, local string) error {
	_, err := c.ExecDevice(ctx, serial, "reverse", remote, local)
	if err != nil {
		return fmt.Errorf("reverse: %w", err)
	}
	return nil
}

func (c *Client) ReverseList(ctx context.Context, serial string) ([]ForwardRule, error) {
	result, err := c.ExecDevice(ctx, serial, "reverse", "--list")
	if err != nil {
		return nil, fmt.Errorf("reverse list: %w", err)
	}
	return parseForwardRules(result.Output), nil
}

func (c *Client) ReverseRemove(ctx context.Context, serial, remote string) error {
	_, err := c.ExecDevice(ctx, serial, "reverse", "--remove", remote)
	if err != nil {
		return fmt.Errorf("reverse remove: %w", err)
	}
	return nil
}

func (c *Client) ReverseRemoveAll(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "reverse", "--remove-all")
	if err != nil {
		return fmt.Errorf("reverse remove all: %w", err)
	}
	return nil
}

func parseForwardRules(output string) []ForwardRule {
	var rules []ForwardRule
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		rules = append(rules, ForwardRule{
			Serial: fields[0],
			Local:  fields[1],
			Remote: fields[2],
		})
	}
	return rules
}
