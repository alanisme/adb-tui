package tui

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

var (
	chmodRe = regexp.MustCompile(`^[0-7]{3,4}$`)
	chownRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+(:[a-zA-Z0-9._-]+)?$`)
)

// fileMessage is implemented by all messages routed to FileModel.
type fileMessage interface{ fileMsg() }

type fileListMsg struct {
	files []adb.FileInfo
	path  string
	err   error
}

type filePushPullMsg struct {
	action string
	err    error
}

type fileDeleteMsg struct {
	err error
}

type filePreviewMsg struct {
	name    string
	content string
	err     error
}

type fileFindMsg struct {
	results []string
	pattern string
	err     error
}

func (fileListMsg) fileMsg()     {}
func (filePushPullMsg) fileMsg() {}
func (fileDeleteMsg) fileMsg()   {}
func (filePreviewMsg) fileMsg()  {}
func (fileFindMsg) fileMsg()     {}

type FileDialogMode int

const (
	FileDialogNone FileDialogMode = iota
	FileDialogPush
	FileDialogPull
	FileDialogMkdir
	FileDialogChmod
	FileDialogChown
	FileDialogFind
)

type FileModel struct {
	client         *adb.Client
	serial         string
	currentDir     string
	files          []adb.FileInfo
	cursor         int
	scroll         int
	width          int
	height         int
	err            error
	statusMsg      string
	loading        bool
	dialog         FileDialogMode
	localInput     textinput.Model
	searchInput    textinput.Model
	showSearch     bool
	searchQuery    string
	confirmDelete  bool
	showBookmarks  bool
	bookmarkIdx    int
	showPreview    bool
	previewContent string
	previewName    string
	previewScroll  int
}

var fileBookmarks = []struct {
	label string
	path  string
}{
	{"SD Card", "/sdcard"},
	{"Downloads", "/sdcard/Download"},
	{"DCIM", "/sdcard/DCIM"},
	{"Pictures", "/sdcard/Pictures"},
	{"Music", "/sdcard/Music"},
	{"Documents", "/sdcard/Documents"},
	{"System", "/system"},
	{"Data", "/data"},
	{"Root", "/"},
}

func NewFileModel(client *adb.Client) FileModel {
	li := textinput.New()
	li.Placeholder = "/local/path"
	li.CharLimit = 256
	si := textinput.New()
	si.Placeholder = "filter files..."
	si.CharLimit = 128
	return FileModel{
		client:      client,
		currentDir:  "/sdcard",
		localInput:  li,
		searchInput: si,
	}
}

func (m FileModel) IsInputCaptured() bool {
	return m.dialog != FileDialogNone || m.showSearch || m.showBookmarks || m.confirmDelete || m.showPreview
}

func (m FileModel) Init() tea.Cmd {
	return nil
}

func (m FileModel) Update(msg tea.Msg) (FileModel, tea.Cmd) {
	switch msg := msg.(type) {
	case fileListMsg:
		m.loading = false
		if msg.err != nil {
			// Navigation failed — show as status, stay in current dir
			m.statusMsg = ErrorStyle.Render("Cannot open: " + msg.err.Error())
		} else {
			m.err = nil
			m.files = msg.files
			m.currentDir = msg.path
			m.cursor = 0
			m.scroll = 0
			m.statusMsg = ""
			m.searchQuery = ""
			m.searchInput.Reset()
		}
		return m, nil

	case filePushPullMsg:
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action + " completed")
		}
		return m, tea.Batch(m.listDir(m.currentDir), clearStatusAfter(5*time.Second))

	case fileDeleteMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = SuccessStyle.Render("Deleted")
		}
		return m, tea.Batch(m.listDir(m.currentDir), clearStatusAfter(5*time.Second))

	case filePreviewMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Preview failed: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.showPreview = true
		m.previewName = msg.name
		m.previewContent = msg.content
		m.previewScroll = 0
		return m, nil

	case fileFindMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Find: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		if len(msg.results) == 0 {
			m.statusMsg = DimStyle.Render("No files found matching: " + msg.pattern)
			return m, clearStatusAfter(5 * time.Second)
		}
		m.showPreview = true
		m.previewName = fmt.Sprintf("Find: %s (%d results)", msg.pattern, len(msg.results))
		m.previewContent = strings.Join(msg.results, "\n")
		m.previewScroll = 0
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showPreview {
			switch msg.String() {
			case "j", "down":
				m.previewScroll++
				lines := strings.Split(m.previewContent, "\n")
				maxScroll := max(len(lines)-m.pageSize(), 0)
				if m.previewScroll > maxScroll {
					m.previewScroll = maxScroll
				}
			case "k", "up":
				m.previewScroll--
				if m.previewScroll < 0 {
					m.previewScroll = 0
				}
			case "esc":
				m.showPreview = false
			}
			return m, nil
		}
		if m.showSearch {
			return m.updateSearch(msg)
		}
		if m.dialog != FileDialogNone {
			return m.updateDialog(msg)
		}
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				m.confirmDelete = false
				if f := m.selectedFile(); f != nil {
					target := path.Join(m.currentDir, f.Name)
					return m, m.deleteFile(target)
				}
			default:
				m.confirmDelete = false
			}
			return m, nil
		}
		if m.showBookmarks {
			switch {
			case key.Matches(msg, DefaultKeyMap.Up):
				if m.bookmarkIdx > 0 {
					m.bookmarkIdx--
				}
			case key.Matches(msg, DefaultKeyMap.Down):
				if m.bookmarkIdx < len(fileBookmarks)-1 {
					m.bookmarkIdx++
				}
			case key.Matches(msg, DefaultKeyMap.Enter):
				m.showBookmarks = false
				target := fileBookmarks[m.bookmarkIdx].path
				m.loading = true
				return m, m.listDir(target)
			case msg.Type == tea.KeyEsc:
				m.showBookmarks = false
			}
			return m, nil
		}
		filteredLen := len(m.filteredFiles())
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < filteredLen-1 {
				m.cursor++
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Enter):
			if f := m.selectedFile(); f != nil {
				if f.IsDir || f.IsLink {
					var target string
					if f.IsLink && f.LinkTarget != "" {
						if path.IsAbs(f.LinkTarget) {
							target = f.LinkTarget
						} else {
							target = path.Join(m.currentDir, f.LinkTarget)
						}
					} else {
						target = path.Join(m.currentDir, f.Name)
					}
					m.loading = true
					return m, m.listDir(target)
				}
			}
		case key.Matches(msg, DefaultKeyMap.Back), msg.Type == tea.KeyBackspace:
			if m.currentDir != "/" {
				parent := path.Dir(m.currentDir)
				m.loading = true
				return m, m.listDir(parent)
			}
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.cursor = 0
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.cursor = max(filteredLen-1, 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.cursor = max(m.cursor-m.pageSize()/2, 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.cursor = min(m.cursor+m.pageSize()/2, max(filteredLen-1, 0))
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.cursor = max(m.cursor-m.pageSize(), 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.cursor = min(m.cursor+m.pageSize(), max(filteredLen-1, 0))
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.listDir(m.currentDir)
		case key.Matches(msg, DefaultKeyMap.Search):
			m.showSearch = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Delete):
			if m.selectedFile() != nil {
				m.confirmDelete = true
			}
		case msg.String() == "p":
			m.dialog = FileDialogPull
			m.localInput.Placeholder = "local destination path"
			m.localInput.Focus()
			return m, textinput.Blink
		case msg.String() == "u":
			m.dialog = FileDialogPush
			m.localInput.Placeholder = "local source path"
			m.localInput.Focus()
			return m, textinput.Blink
		case msg.String() == "b":
			m.showBookmarks = true
			m.bookmarkIdx = 0
		case msg.String() == " ":
			if f := m.selectedFile(); f != nil && !f.IsDir {
				remotePath := path.Join(m.currentDir, f.Name)
				return m, m.previewFile(*f, remotePath)
			}
		case msg.String() == "m":
			m.dialog = FileDialogMkdir
			m.localInput.Placeholder = "directory name"
			m.localInput.Focus()
			return m, textinput.Blink
		case msg.String() == "F":
			m.dialog = FileDialogFind
			m.localInput.Placeholder = "search pattern (e.g. *.txt)"
			m.localInput.Focus()
			return m, textinput.Blink
		case msg.String() == "c":
			if m.selectedFile() != nil {
				m.dialog = FileDialogChmod
				m.localInput.Placeholder = "permission mode (e.g. 755)"
				m.localInput.Focus()
				return m, textinput.Blink
			}
		case msg.String() == "o":
			if m.selectedFile() != nil {
				m.dialog = FileDialogChown
				m.localInput.Placeholder = "owner[:group] (e.g. root:root)"
				m.localInput.Focus()
				return m, textinput.Blink
			}
		}
	}
	return m, nil
}

func (m FileModel) updateDialog(msg tea.KeyMsg) (FileModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		localPath := m.localInput.Value()
		dialog := m.dialog
		m.dialog = FileDialogNone
		m.localInput.Reset()
		m.localInput.Blur()
		if localPath == "" {
			return m, nil
		}
		switch dialog {
		case FileDialogPull:
			if f := m.selectedFile(); f != nil {
				remotePath := path.Join(m.currentDir, f.Name)
				return m, m.pullFile(remotePath, localPath)
			}
		case FileDialogPush:
			return m, m.pushFile(localPath, path.Clean(m.currentDir)+"/")
		case FileDialogMkdir:
			dirPath := path.Join(m.currentDir, localPath)
			return m, m.mkdirCmd(dirPath)
		case FileDialogChmod:
			if !chmodRe.MatchString(localPath) {
				m.statusMsg = ErrorStyle.Render("Invalid mode: use octal like 755 or 0644")
				return m, clearStatusAfter(5 * time.Second)
			}
			if f := m.selectedFile(); f != nil {
				target := path.Join(m.currentDir, f.Name)
				return m, m.chmodCmd(target, localPath)
			}
		case FileDialogChown:
			if !chownRe.MatchString(localPath) {
				m.statusMsg = ErrorStyle.Render("Invalid owner: use owner or owner:group")
				return m, clearStatusAfter(5 * time.Second)
			}
			if f := m.selectedFile(); f != nil {
				target := path.Join(m.currentDir, f.Name)
				return m, m.chownCmd(target, localPath)
			}
		case FileDialogFind:
			return m, m.findCmd(m.currentDir, localPath)
		}
		return m, nil
	case tea.KeyEsc:
		m.dialog = FileDialogNone
		m.localInput.Reset()
		m.localInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.localInput, cmd = m.localInput.Update(msg)
	return m, cmd
}

func (m FileModel) updateSearch(msg tea.KeyMsg) (FileModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searchQuery = m.searchInput.Value()
		m.showSearch = false
		m.searchInput.Blur()
		m.cursor = 0
		m.scroll = 0
		return m, nil
	case tea.KeyEsc:
		m.searchQuery = ""
		m.showSearch = false
		m.searchInput.Reset()
		m.searchInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m FileModel) filteredFiles() []int {
	if m.searchQuery == "" {
		indices := make([]int, len(m.files))
		for i := range indices {
			indices[i] = i
		}
		return indices
	}
	q := strings.ToLower(m.searchQuery)
	var indices []int
	for i, f := range m.files {
		if strings.Contains(strings.ToLower(f.Name), q) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m FileModel) View() string {
	var b strings.Builder

	filtered := m.filteredFiles()
	title := fmt.Sprintf("  Files %s  %s",
		scrollInfo(m.cursor, len(filtered)),
		AccentStyle.Render(m.currentDir))
	b.WriteString(HeaderStyle.Render(title))
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.loading {
		b.WriteString(DimStyle.Render("  Loading...") + "\n")
		return b.String()
	}

	if m.showPreview {
		b.WriteString("  " + AccentStyle.Render("Preview: "+m.previewName) + "\n")
		b.WriteString(DimStyle.Render("  "+strings.Repeat("─", max(m.width-4, 20))) + "\n")

		lines := strings.Split(m.previewContent, "\n")
		viewHeight := safeViewHeight(m.height, 10, 15)
		end := min(m.previewScroll+viewHeight, len(lines))
		for i := m.previewScroll; i < end; i++ {
			fmt.Fprintf(&b, "  %s %s\n", DimStyle.Render(fmt.Sprintf("%4d", i+1)), lines[i])
		}

		b.WriteString("\n" + helpBar(
			keyHint("j/k", "scroll"),
			keyHint("esc", "close"),
		))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	header := fmt.Sprintf("  %-10s %-10s %-16s %s", "Perms", "Size", "Modified", "Name")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	viewHeight := safeViewHeight(m.height, 10, 15)

	if m.searchQuery != "" {
		b.WriteString("  " + DimStyle.Render(fmt.Sprintf("Filter: %s (%d matches)", m.searchQuery, len(filtered))) + "\n")
	}

	end := min(m.scroll+viewHeight, len(filtered))

	for i := m.scroll; i < end; i++ {
		f := m.files[filtered[i]]
		prefix := "  "
		style := NormalStyle
		if i == m.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}

		icon := " "
		if f.IsDir {
			icon = "/"
		} else if f.IsLink {
			icon = "@"
		}

		sizeStr := formatFileSize(f.Size)
		dateStr := ""
		if !f.ModTime.IsZero() {
			dateStr = f.ModTime.Format("2006-01-02 15:04")
		}

		line := fmt.Sprintf("%-10s %10s %-16s %s%s",
			DimStyle.Render(f.Permissions),
			DimStyle.Render(sizeStr),
			DimStyle.Render(dateStr),
			style.Render(f.Name),
			DimStyle.Render(icon),
		)
		b.WriteString(prefix + line + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	if m.dialog != FileDialogNone {
		var label string
		switch m.dialog {
		case FileDialogPull:
			label = "Pull to: "
		case FileDialogPush:
			label = "Push from: "
		case FileDialogMkdir:
			label = "Directory name: "
		case FileDialogChmod:
			label = "Mode: "
		case FileDialogChown:
			label = "Owner: "
		case FileDialogFind:
			label = "Find: "
		}
		b.WriteString("\n  " + DialogStyle.Render(label+m.localInput.View()) + "\n")
	}

	if m.showSearch {
		b.WriteString("\n  " + m.searchInput.View() + "\n")
	}

	if m.confirmDelete {
		if f := m.selectedFile(); f != nil {
			b.WriteString("\n" + DialogStyle.Render(
				fmt.Sprintf("Delete %q? [y/N]", f.Name)) + "\n")
		}
	}

	if m.showBookmarks {
		var bm strings.Builder
		bm.WriteString("Bookmarks:\n")
		for i, bk := range fileBookmarks {
			prefix := "  "
			if i == m.bookmarkIdx {
				prefix = "▸ "
			}
			fmt.Fprintf(&bm, "%s%-12s %s\n", prefix, bk.label, DimStyle.Render(bk.path))
		}
		b.WriteString("\n" + DialogStyle.Render(bm.String()) + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("⏎", "open"),
		keyHint("spc", "preview"),
		keyHint("esc/⌫", "parent"),
		keyHint("/", "search"),
		keyHint("b", "bookmarks"),
		keyHint("p", "pull"),
		keyHint("u", "push"),
		keyHint("d", "delete"),
		keyHint("m", "mkdir"),
		keyHint("F", "find"),
		keyHint("c", "chmod"),
		keyHint("o", "chown"),
	))

	return b.String()
}

func (m FileModel) pageSize() int {
	return max(m.height-10, 10)
}

func (m FileModel) selectedFile() *adb.FileInfo {
	filtered := m.filteredFiles()
	if m.cursor >= 0 && m.cursor < len(filtered) {
		return &m.files[filtered[m.cursor]]
	}
	return nil
}

func (m *FileModel) ensureVisible() {
	viewHeight := safeViewHeight(m.height, 10, 15)
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+viewHeight {
		m.scroll = m.cursor - viewHeight + 1
	}
}

func (m FileModel) SetDevice(serial string) (FileModel, tea.Cmd) {
	m.serial = serial
	m.currentDir = "/sdcard"
	m.files = nil
	m.cursor = 0
	m.scroll = 0
	if serial == "" {
		return m, nil
	}
	m.loading = true
	return m, m.listDir(m.currentDir)
}

func (m FileModel) SetSize(w, h int) FileModel {
	m.width = w
	m.height = h
	return m
}

func (m FileModel) listDir(path string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		files, err := client.ListDir(ctx, serial, path)
		return fileListMsg{files: files, path: path, err: err}
	}
}

func (m FileModel) pullFile(remote, local string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := client.Pull(ctx, serial, remote, local)
		return filePushPullMsg{action: "Pull", err: err}
	}
}

func (m FileModel) pushFile(local, remote string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := client.Push(ctx, serial, local, remote)
		return filePushPullMsg{action: "Push", err: err}
	}
}

var binaryExtensions = map[string]bool{
	".apk": true, ".aab": true, ".dex": true, ".so": true, ".o": true, ".a": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".webp": true, ".ico": true,
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".3gp": true, ".webm": true,
	".mp3": true, ".ogg": true, ".wav": true, ".flac": true, ".aac": true, ".m4a": true,
	".zip": true, ".gz": true, ".tar": true, ".bz2": true, ".xz": true, ".rar": true, ".7z": true,
	".db": true, ".sqlite": true, ".sqlite3": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".class": true, ".jar": true, ".aar": true,
}

func isBinaryFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return binaryExtensions[ext]
}

const maxPreviewSize = 1 << 20 // 1 MB

func (m FileModel) previewFile(f adb.FileInfo, remotePath string) tea.Cmd {
	if isBinaryFile(f.Name) {
		return func() tea.Msg {
			return filePreviewMsg{name: f.Name, err: fmt.Errorf("binary file, preview not supported")}
		}
	}
	if f.Size > maxPreviewSize {
		return func() tea.Msg {
			return filePreviewMsg{name: f.Name, err: fmt.Errorf("file too large (%s), preview limit is 1MB", formatFileSize(f.Size))}
		}
	}
	if f.IsLink {
		return func() tea.Msg {
			return filePreviewMsg{name: f.Name, err: fmt.Errorf("symlink preview not supported (target: %s)", f.LinkTarget)}
		}
	}
	serial := m.serial
	client := m.client
	name := f.Name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		content, err := client.Head(ctx, serial, remotePath, 100)
		if err != nil {
			return filePreviewMsg{name: name, err: err}
		}
		// Detect binary content (NUL bytes)
		if strings.ContainsRune(content, '\x00') {
			return filePreviewMsg{name: name, err: fmt.Errorf("binary file detected, preview not supported")}
		}
		lines := strings.SplitN(content, "\n", 101)
		if len(lines) > 100 {
			lines = lines[:100]
			content = strings.Join(lines, "\n") + "\n…(truncated)"
		}
		return filePreviewMsg{name: name, content: content}
	}
}

func (m FileModel) deleteFile(path string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.Remove(ctx, serial, path)
		return fileDeleteMsg{err: err}
	}
}

func (m FileModel) mkdirCmd(dirPath string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.Mkdir(ctx, serial, dirPath)
		return filePushPullMsg{action: "Mkdir", err: err}
	}
}

func (m FileModel) chmodCmd(target, mode string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.Chmod(ctx, serial, target, mode)
		return filePushPullMsg{action: "Chmod " + mode, err: err}
	}
}

func (m FileModel) chownCmd(target, owner string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.Chown(ctx, serial, target, owner)
		return filePushPullMsg{action: "Chown " + owner, err: err}
	}
}

func (m FileModel) findCmd(dir, pattern string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		results, err := client.Find(ctx, serial, dir, pattern)
		return fileFindMsg{results: results, pattern: pattern, err: err}
	}
}

func formatFileSize(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", size)
	}
}
