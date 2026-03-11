package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FileInfo struct {
	Name        string
	Size        int64
	Permissions string
	ModTime     time.Time
	IsDir       bool
	IsLink      bool
	LinkTarget  string
}

func (c *Client) Push(ctx context.Context, serial, local, remote string) error {
	_, err := c.ExecDevice(ctx, serial, "push", local, remote)
	if err != nil {
		return fmt.Errorf("push %s to %s: %w", local, remote, err)
	}
	return nil
}

func (c *Client) Pull(ctx context.Context, serial, remote, local string) error {
	_, err := c.ExecDevice(ctx, serial, "pull", remote, local)
	if err != nil {
		return fmt.Errorf("pull %s to %s: %w", remote, local, err)
	}
	return nil
}

func (c *Client) ListDir(ctx context.Context, serial, dirPath string) ([]FileInfo, error) {
	// Use ls -laF for type indicators: / for dirs, @ for links, * for executables
	result, err := c.Shell(ctx, serial, "ls -la "+shellQuote(dirPath)+" 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("list dir %s: %w", dirPath, err)
	}

	var files []FileInfo
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}
		fi, ok := parseLsLine(line)
		if ok && fi.Name != "." && fi.Name != ".." {
			files = append(files, fi)
		}
	}
	return files, nil
}

func parseLsLine(line string) (FileInfo, bool) {
	fields := strings.Fields(line)
	if len(fields) < 7 {
		return FileInfo{}, false
	}

	perms := fields[0]
	fi := FileInfo{
		Permissions: perms,
		IsDir:       len(perms) > 0 && perms[0] == 'd',
		IsLink:      len(perms) > 0 && perms[0] == 'l',
	}

	// Size field position varies; try to find date pattern to anchor parsing.
	// Typical format: perms links owner group size YYYY-MM-DD HH:MM name
	dateIdx := -1
	for i := 4; i < len(fields)-1; i++ {
		if len(fields[i]) == 10 && fields[i][4] == '-' && fields[i][7] == '-' {
			dateIdx = i
			break
		}
	}

	if dateIdx >= 0 && dateIdx+2 < len(fields) {
		if dateIdx > 0 {
			size, err := strconv.ParseInt(fields[dateIdx-1], 10, 64)
			if err == nil {
				fi.Size = size
			}
		}

		dateStr := fields[dateIdx] + " " + fields[dateIdx+1]
		if t, err := time.Parse("2006-01-02 15:04", dateStr); err == nil {
			fi.ModTime = t
		}

		nameIdx := dateIdx + 2
		remaining := strings.Join(fields[nameIdx:], " ")
		if fi.IsLink {
			if before, after, ok := strings.Cut(remaining, " -> "); ok {
				fi.Name = before
				fi.LinkTarget = after
			} else {
				fi.Name = remaining
			}
		} else {
			fi.Name = remaining
		}
	} else {
		fi.Name = fields[len(fields)-1]
	}

	return fi, fi.Name != ""
}

func (c *Client) Stat(ctx context.Context, serial, path string) (*FileInfo, error) {
	result, err := c.ShellArgs(ctx, serial, "ls", "-la", "-d", path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	line := strings.TrimSpace(result.Output)
	fi, ok := parseLsLine(line)
	if !ok {
		return nil, fmt.Errorf("stat %s: unable to parse output", path)
	}
	return &fi, nil
}

func (c *Client) Remove(ctx context.Context, serial, path string) error {
	_, err := c.ShellArgs(ctx, serial, "rm", "-rf", path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

func (c *Client) Mkdir(ctx context.Context, serial, path string) error {
	_, err := c.ShellArgs(ctx, serial, "mkdir", "-p", path)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

func (c *Client) Cat(ctx context.Context, serial, path string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "cat", path)
	if err != nil {
		return "", fmt.Errorf("cat %s: %w", path, err)
	}
	return result.Output, nil
}

// Head reads the first n lines of a file safely using ShellArgs.
func (c *Client) Head(ctx context.Context, serial, path string, lines int) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "head", "-n", strconv.Itoa(lines), path)
	if err != nil {
		return "", fmt.Errorf("head %s: %w", path, err)
	}
	return result.Output, nil
}

func (c *Client) Chmod(ctx context.Context, serial, path, mode string) error {
	_, err := c.ShellArgs(ctx, serial, "chmod", mode, path)
	if err != nil {
		return fmt.Errorf("chmod %s %s: %w", mode, path, err)
	}
	return nil
}

func (c *Client) Chown(ctx context.Context, serial, path, owner string) error {
	_, err := c.ShellArgs(ctx, serial, "chown", owner, path)
	if err != nil {
		return fmt.Errorf("chown %s %s: %w", owner, path, err)
	}
	return nil
}

func (c *Client) Find(ctx context.Context, serial, path, name string) ([]string, error) {
	result, err := c.ShellArgs(ctx, serial, "find", path, "-name", name)
	if err != nil {
		return nil, fmt.Errorf("find %s -name %s: %w", path, name, err)
	}

	var files []string
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func (c *Client) DiskFree(ctx context.Context, serial string) ([]DiskUsage, error) {
	return c.GetDiskUsage(ctx, serial)
}
