package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// packageMessage is implemented by all messages routed to PackageModel.
type packageMessage interface{ packageMsg() }

type packageListMsg struct {
	packages []adb.PackageInfo
	err      error
}

type packageDetailMsg struct {
	detail *adb.PackageDetail
	err    error
}

type packageActionMsg struct {
	action string
	err    error
}

type activityStackMsg struct {
	pkg        string
	activities []string
	err        error
}

type permissionListMsg struct {
	pkg         string
	permissions []adb.PermissionInfo
	err         error
}

type batchInstallMsg struct {
	total     int
	succeeded int
	failed    int
	errors    []string
}

func (packageListMsg) packageMsg()    {}
func (packageDetailMsg) packageMsg()  {}
func (packageActionMsg) packageMsg()  {}
func (activityStackMsg) packageMsg()  {}
func (permissionListMsg) packageMsg() {}
func (batchInstallMsg) packageMsg()   {}

type PackageFilter int

const (
	PackageFilterAll PackageFilter = iota
	PackageFilterThirdParty
	PackageFilterSystem
)

type PackageModel struct {
	client           *adb.Client
	serial           string
	packages         []adb.PackageInfo
	visible          []int
	cursor           int
	scroll           int
	width            int
	height           int
	err              error
	loading          bool
	filter           PackageFilter
	searchInput      textinput.Model
	showSearch       bool
	searchQuery      string
	detail           *adb.PackageDetail
	showDetail       bool
	statusMsg        string
	installInput     textinput.Model
	showInstall      bool
	batchInput       textinput.Model
	showBatchInstall bool
	confirmAction    string
	confirmPkg       string

	// Activity stack
	showActivities bool
	activities     []string
	activityPkg    string
	activityScroll int

	// Permissions
	showPermissions bool
	permissions     []adb.PermissionInfo
	permPkg         string
	permCursor      int
	permScroll      int
	permConfirm     string // "grant" or "revoke"
	permConfirmName string
}

func NewPackageModel(client *adb.Client) PackageModel {
	si := textinput.New()
	si.Placeholder = "filter packages..."
	si.CharLimit = 128

	ii := textinput.New()
	ii.Placeholder = "/path/to/app.apk"
	ii.CharLimit = 256

	bi := textinput.New()
	bi.Placeholder = "/path/to/apks/ (directory containing .apk files)"
	bi.CharLimit = 256

	return PackageModel{
		client:       client,
		filter:       PackageFilterAll,
		searchInput:  si,
		installInput: ii,
		batchInput:   bi,
	}
}

func (m PackageModel) IsInputCaptured() bool {
	return m.showSearch || m.showInstall || m.showBatchInstall || m.showDetail || m.confirmAction != "" || m.showActivities || m.showPermissions
}

func (m PackageModel) Init() tea.Cmd {
	return nil
}

func (m PackageModel) Update(msg tea.Msg) (PackageModel, tea.Cmd) {
	switch msg := msg.(type) {
	case packageListMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.packages = msg.packages
			m.rebuildVisible()
		}
		return m, nil

	case packageDetailMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.detail = msg.detail
			m.showDetail = true
		}
		return m, nil

	case activityStackMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Activities: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.activities = msg.activities
		m.activityPkg = msg.pkg
		m.showActivities = true
		m.activityScroll = 0
		return m, nil

	case permissionListMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Permissions: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.permissions = msg.permissions
		m.permPkg = msg.pkg
		m.showPermissions = true
		m.permCursor = 0
		m.permScroll = 0
		return m, nil

	case packageActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else if strings.HasPrefix(msg.action, "APK: ") {
			m.statusMsg = SuccessStyle.Render(msg.action)
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action + " completed")
		}
		return m, tea.Batch(m.fetchPackages(), clearStatusAfter(5*time.Second))

	case batchInstallMsg:
		if msg.total == 0 && len(msg.errors) > 0 {
			// Directory read error — show the real error, not "no APKs"
			m.statusMsg = ErrorStyle.Render(strings.Join(msg.errors, "; "))
		} else if msg.total == 0 {
			m.statusMsg = WarningStyle.Render("No .apk files found in directory")
		} else if msg.failed == 0 {
			m.statusMsg = SuccessStyle.Render(fmt.Sprintf("Batch install: %d/%d succeeded", msg.succeeded, msg.total))
		} else {
			summary := fmt.Sprintf("Batch install: %d/%d succeeded, %d failed", msg.succeeded, msg.total, msg.failed)
			if len(msg.errors) > 0 {
				summary += "\n" + strings.Join(msg.errors, "\n")
			}
			m.statusMsg = WarningStyle.Render(summary)
		}
		return m, tea.Batch(m.fetchPackages(), clearStatusAfter(10*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showSearch {
			return m.updateSearch(msg)
		}
		if m.showInstall {
			return m.updateInstall(msg)
		}
		if m.showBatchInstall {
			return m.updateBatchInstall(msg)
		}
		if m.showActivities {
			return m.updateActivities(msg)
		}
		if m.showPermissions {
			return m.updatePermissions(msg)
		}
		if m.showDetail {
			switch {
			case msg.Type == tea.KeyEsc:
				m.showDetail = false
				m.detail = nil
			case msg.String() == "a":
				if m.detail != nil {
					return m, m.fetchActivities(m.detail.Name)
				}
			case msg.String() == "P":
				if m.detail != nil {
					return m, m.fetchPermissions(m.detail.Name)
				}
			}
			return m, nil
		}
		if m.confirmAction != "" {
			switch msg.String() {
			case "y", "Y":
				action := m.confirmAction
				pkg := m.confirmPkg
				m.confirmAction = ""
				m.confirmPkg = ""
				switch action {
				case "uninstall":
					return m, m.uninstallPackage(pkg)
				case "clear-data":
					return m, m.clearData(pkg)
				}
			default:
				m.confirmAction = ""
				m.confirmPkg = ""
			}
			return m, nil
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Enter):
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.fetchDetail(pkg.Name)
			}
		case key.Matches(msg, DefaultKeyMap.Search):
			m.showSearch = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Filter):
			m.filter = (m.filter + 1) % 3
			m.loading = true
			return m, m.fetchPackages()
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.fetchPackages()
		case key.Matches(msg, DefaultKeyMap.Install):
			m.showInstall = true
			m.installInput.Focus()
			return m, textinput.Blink
		case msg.String() == "I":
			m.showBatchInstall = true
			m.batchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.cursor = 0
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.cursor = max(len(m.visible)-1, 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.cursor = max(m.cursor-m.pageSize()/2, 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.cursor = min(m.cursor+m.pageSize()/2, max(len(m.visible)-1, 0))
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.cursor = max(m.cursor-m.pageSize(), 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.cursor = min(m.cursor+m.pageSize(), max(len(m.visible)-1, 0))
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.Delete):
			if pkg := m.selectedPackage(); pkg != nil {
				m.confirmAction = "uninstall"
				m.confirmPkg = pkg.Name
			}
		case msg.String() == "c":
			if pkg := m.selectedPackage(); pkg != nil {
				m.confirmAction = "clear-data"
				m.confirmPkg = pkg.Name
			}
		case msg.String() == "x":
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.forceStop(pkg.Name)
			}
		case msg.String() == "e":
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.enablePackage(pkg.Name)
			}
		case msg.String() == "w":
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.disablePackage(pkg.Name)
			}
		case msg.String() == "o":
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.launchApp(pkg.Name)
			}
		case msg.String() == "p":
			if pkg := m.selectedPackage(); pkg != nil {
				return m, m.showAPKPath(pkg.Name)
			}
		}
	}
	return m, nil
}

func (m PackageModel) updateSearch(msg tea.KeyMsg) (PackageModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searchQuery = m.searchInput.Value()
		m.showSearch = false
		m.searchInput.Blur()
		m.rebuildVisible()
		return m, nil
	case tea.KeyEsc:
		m.showSearch = false
		m.searchInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m PackageModel) updateInstall(msg tea.KeyMsg) (PackageModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		path := m.installInput.Value()
		m.showInstall = false
		m.installInput.Reset()
		m.installInput.Blur()
		if path != "" {
			return m, m.installAPK(path)
		}
		return m, nil
	case tea.KeyEsc:
		m.showInstall = false
		m.installInput.Reset()
		m.installInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.installInput, cmd = m.installInput.Update(msg)
	return m, cmd
}

func (m PackageModel) updateBatchInstall(msg tea.KeyMsg) (PackageModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		dir := m.batchInput.Value()
		m.showBatchInstall = false
		m.batchInput.Reset()
		m.batchInput.Blur()
		if dir != "" {
			m.statusMsg = DimStyle.Render("Batch installing from " + dir + "...")
			return m, m.batchInstallAPKs(dir)
		}
		return m, nil
	case tea.KeyEsc:
		m.showBatchInstall = false
		m.batchInput.Reset()
		m.batchInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.batchInput, cmd = m.batchInput.Update(msg)
	return m, cmd
}

func (m PackageModel) batchInstallAPKs(dir string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return batchInstallMsg{errors: []string{"Read directory: " + err.Error()}}
		}

		var apks []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".apk") {
				apks = append(apks, filepath.Join(dir, e.Name()))
			}
		}

		if len(apks) == 0 {
			return batchInstallMsg{}
		}

		var succeeded, failed int
		var errors []string
		for _, path := range apks {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			installErr := client.InstallAPK(ctx, serial, path, adb.InstallOptions{Reinstall: true})
			cancel()
			if installErr != nil {
				failed++
				errors = append(errors, filepath.Base(path)+": "+installErr.Error())
			} else {
				succeeded++
			}
		}
		return batchInstallMsg{
			total:     len(apks),
			succeeded: succeeded,
			failed:    failed,
			errors:    errors,
		}
	}
}

func (m PackageModel) View() string {
	var b strings.Builder

	filterLabel := "all"
	switch m.filter {
	case PackageFilterThirdParty:
		filterLabel = "third-party"
	case PackageFilterSystem:
		filterLabel = "system"
	}

	title := fmt.Sprintf("  Packages %s  %s  total:%d",
		scrollInfo(m.cursor, len(m.visible)),
		AccentStyle.Render(filterLabel),
		len(m.packages))
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

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	if m.showActivities {
		b.WriteString(m.renderActivities())
		return b.String()
	}

	if m.showPermissions {
		b.WriteString(m.renderPermissions())
		return b.String()
	}

	if m.showDetail && m.detail != nil {
		b.WriteString(m.renderDetail())
		return b.String()
	}

	if m.searchQuery != "" {
		b.WriteString("  " + DimStyle.Render("Search: "+m.searchQuery) + "\n")
	}

	viewHeight := safeViewHeight(m.height, 10, 15)

	end := min(m.scroll+viewHeight, len(m.visible))

	for i := m.scroll; i < end; i++ {
		idx := m.visible[i]
		pkg := m.packages[idx]
		prefix := "  "
		style := NormalStyle
		if i == m.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		b.WriteString(prefix + style.Render(pkg.Name) + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	if m.showSearch {
		b.WriteString("\n  " + m.searchInput.View() + "\n")
	}
	if m.showInstall {
		b.WriteString("\n" + DialogStyle.Render("Install APK: "+m.installInput.View()) + "\n")
	}
	if m.showBatchInstall {
		b.WriteString("\n" + DialogStyle.Render("Batch Install (directory): "+m.batchInput.View()) + "\n")
	}
	if m.confirmAction != "" {
		label := "Uninstall"
		if m.confirmAction == "clear-data" {
			label = "Clear data for"
		}
		b.WriteString("\n" + DialogStyle.Render(
			fmt.Sprintf("%s %s? [y/N]", label, m.confirmPkg)) + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("/", "search"),
		keyHint("f", "filter"),
		keyHint("i", "install"),
		keyHint("I", "batch"),
		keyHint("d", "uninstall"),
		keyHint("o", "open"),
		keyHint("p", "apk path"),
		keyHint("x", "stop"),
		keyHint("c", "clear"),
	))

	return b.String()
}

func (m PackageModel) renderDetail() string {
	var b strings.Builder
	d := m.detail
	b.WriteString("\n")
	b.WriteString("  " + TitleStyle.Render(d.Name) + "\n\n")

	fields := []struct {
		label string
		value string
	}{
		{"Version", d.VersionName + " (" + d.VersionCode + ")"},
		{"Installer", d.Installer},
		{"UID", d.UID},
		{"APK Path", d.APKPath},
		{"Data Dir", d.DataDir},
		{"First Install", d.FirstInstall},
		{"Last Update", d.LastUpdate},
		{"System", fmt.Sprintf("%v", d.System)},
		{"Enabled", fmt.Sprintf("%v", d.Enabled)},
	}

	for _, f := range fields {
		if f.value != "" && f.value != " ()" {
			fmt.Fprintf(&b, "    %-16s %s\n", AccentStyle.Render(f.label), f.value)
		}
	}

	if len(d.Permissions) > 0 {
		b.WriteString("\n  " + AccentStyle.Render("Permissions:") + "\n")
		for _, p := range d.Permissions {
			b.WriteString("    " + DimStyle.Render(p) + "\n")
		}
	}

	b.WriteString("\n" + helpBar(
		keyHint("a", "activities"),
		keyHint("P", "permissions"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func (m *PackageModel) rebuildVisible() {
	m.visible = nil
	q := strings.ToLower(m.searchQuery)
	for i, pkg := range m.packages {
		if m.searchQuery != "" && !strings.Contains(strings.ToLower(pkg.Name), q) {
			continue
		}
		m.visible = append(m.visible, i)
	}
	m.cursor = 0
	m.scroll = 0
}

func (m *PackageModel) ensureVisible() {
	viewHeight := safeViewHeight(m.height, 10, 15)
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+viewHeight {
		m.scroll = m.cursor - viewHeight + 1
	}
}

func (m PackageModel) pageSize() int {
	return safeViewHeight(m.height, 10, 10)
}

func (m PackageModel) selectedPackage() *adb.PackageInfo {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		idx := m.visible[m.cursor]
		return &m.packages[idx]
	}
	return nil
}

func (m PackageModel) SetDevice(serial string) (PackageModel, tea.Cmd) {
	m.serial = serial
	m.packages = nil
	m.visible = nil
	m.cursor = 0
	m.scroll = 0
	m.showDetail = false
	m.detail = nil
	if serial == "" {
		return m, nil
	}
	m.loading = true
	return m, m.fetchPackages()
}

func (m PackageModel) SetSize(w, h int) PackageModel {
	m.width = w
	m.height = h
	return m
}

func (m PackageModel) fetchPackages() tea.Cmd {
	serial := m.serial
	client := m.client
	filter := m.filter
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		opts := adb.ListOptions{}
		switch filter {
		case PackageFilterThirdParty:
			opts.ShowThirdParty = true
		case PackageFilterSystem:
			opts.ShowSystem = true
		}
		packages, err := client.ListPackages(ctx, serial, opts)
		return packageListMsg{packages: packages, err: err}
	}
}

func (m PackageModel) fetchDetail(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		detail, err := client.GetPackageInfo(ctx, serial, pkg)
		return packageDetailMsg{detail: detail, err: err}
	}
}

func (m PackageModel) installAPK(path string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		err := client.InstallAPK(ctx, serial, path, adb.InstallOptions{Reinstall: true})
		return packageActionMsg{action: "Install", err: err}
	}
}

func (m PackageModel) uninstallPackage(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := client.UninstallPackage(ctx, serial, pkg, false)
		return packageActionMsg{action: "Uninstall", err: err}
	}
}

func (m PackageModel) clearData(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.ClearData(ctx, serial, pkg)
		return packageActionMsg{action: "Clear data", err: err}
	}
}

func (m PackageModel) forceStop(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.ForceStop(ctx, serial, pkg)
		return packageActionMsg{action: "Force stop", err: err}
	}
}

func (m PackageModel) enablePackage(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.EnablePackage(ctx, serial, pkg)
		return packageActionMsg{action: "Enable", err: err}
	}
}

func (m PackageModel) launchApp(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := client.ShellArgs(ctx, serial, "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
		return packageActionMsg{action: "Launch " + pkg, err: err}
	}
}

func (m PackageModel) showAPKPath(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		path, err := client.GetAPKPath(ctx, serial, pkg)
		if err != nil {
			return packageActionMsg{action: "APK path", err: err}
		}
		return packageActionMsg{action: fmt.Sprintf("APK: %s", path), err: nil}
	}
}

func (m PackageModel) disablePackage(pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.DisablePackage(ctx, serial, pkg)
		return packageActionMsg{action: "Disable", err: err}
	}
}

// Activity stack

func (m PackageModel) updateActivities(msg tea.KeyMsg) (PackageModel, tea.Cmd) {
	maxVisible := safeViewHeight(m.height, 8, 5)
	maxScroll := max(len(m.activities)-maxVisible, 0)
	switch {
	case msg.Type == tea.KeyEsc:
		m.showActivities = false
	case key.Matches(msg, DefaultKeyMap.Up):
		m.activityScroll = max(m.activityScroll-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		m.activityScroll = min(m.activityScroll+1, maxScroll)
	}
	return m, nil
}

func (m PackageModel) fetchActivities(pkg string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		activities, err := client.ListActivities(ctx, serial, pkg)
		return activityStackMsg{pkg: pkg, activities: activities, err: err}
	}
}

func (m PackageModel) renderActivities() string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Activities: "+m.activityPkg) + "\n\n")

	if len(m.activities) == 0 {
		b.WriteString("  " + DimStyle.Render("No activities found") + "\n")
	} else {
		maxVisible := safeViewHeight(m.height, 8, 5)
		end := min(m.activityScroll+maxVisible, len(m.activities))
		for i, act := range m.activities[m.activityScroll:end] {
			idx := m.activityScroll + i
			fmt.Fprintf(&b, "  %s %s\n", DimStyle.Render(fmt.Sprintf("%3d", idx+1)), act)
		}
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "scroll"),
		keyHint("esc", "back"),
	))
	return b.String()
}

// Permissions management

func (m PackageModel) updatePermissions(msg tea.KeyMsg) (PackageModel, tea.Cmd) {
	if m.permConfirm != "" {
		switch msg.String() {
		case "y", "Y":
			action := m.permConfirm
			perm := m.permConfirmName
			m.permConfirm = ""
			m.permConfirmName = ""
			if action == "grant" {
				return m, m.grantPerm(m.permPkg, perm)
			}
			return m, m.revokePerm(m.permPkg, perm)
		default:
			m.permConfirm = ""
			m.permConfirmName = ""
		}
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyEsc:
		m.showPermissions = false
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.permCursor > 0 {
			m.permCursor--
			m.ensurePermVisible()
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.permCursor < len(m.permissions)-1 {
			m.permCursor++
			m.ensurePermVisible()
		}
	case msg.String() == "G":
		if p := m.selectedPerm(); p != nil && !p.Granted {
			m.permConfirm = "grant"
			m.permConfirmName = p.Name
		}
	case msg.String() == "R":
		if p := m.selectedPerm(); p != nil && p.Granted {
			m.permConfirm = "revoke"
			m.permConfirmName = p.Name
		}
	case key.Matches(msg, DefaultKeyMap.Refresh):
		return m, m.fetchPermissions(m.permPkg)
	}
	return m, nil
}

func (m PackageModel) selectedPerm() *adb.PermissionInfo {
	if m.permCursor >= 0 && m.permCursor < len(m.permissions) {
		return &m.permissions[m.permCursor]
	}
	return nil
}

func (m *PackageModel) ensurePermVisible() {
	viewH := safeViewHeight(m.height, 10, 5)
	if m.permCursor < m.permScroll {
		m.permScroll = m.permCursor
	}
	if m.permCursor >= m.permScroll+viewH {
		m.permScroll = m.permCursor - viewH + 1
	}
}

func (m PackageModel) fetchPermissions(pkg string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		perms, err := client.ListAppPermissions(ctx, serial, pkg)
		return permissionListMsg{pkg: pkg, permissions: perms, err: err}
	}
}

func (m PackageModel) grantPerm(pkg, perm string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.GrantPermission(ctx, serial, pkg, perm)
		if err != nil {
			return packageActionMsg{action: "Grant " + perm, err: err}
		}
		// Refresh permissions with a fresh context
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		perms, err := client.ListAppPermissions(ctx2, serial, pkg)
		return permissionListMsg{pkg: pkg, permissions: perms, err: err}
	}
}

func (m PackageModel) revokePerm(pkg, perm string) tea.Cmd {
	serial, client := m.serial, m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.RevokePermission(ctx, serial, pkg, perm)
		if err != nil {
			return packageActionMsg{action: "Revoke " + perm, err: err}
		}
		// Refresh permissions with a fresh context
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		perms, err := client.ListAppPermissions(ctx2, serial, pkg)
		return permissionListMsg{pkg: pkg, permissions: perms, err: err}
	}
}

func (m PackageModel) renderPermissions() string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Permissions: "+m.permPkg) + "\n\n")

	if len(m.permissions) == 0 {
		b.WriteString("  " + DimStyle.Render("No permissions found") + "\n")
	} else {
		viewH := safeViewHeight(m.height, 10, 5)
		end := min(m.permScroll+viewH, len(m.permissions))
		for i := m.permScroll; i < end; i++ {
			p := m.permissions[i]
			prefix := "  "
			style := NormalStyle
			if i == m.permCursor {
				prefix = CursorStyle.Render("▸ ")
				style = SelectedStyle
			}
			status := DimStyle.Render("✗")
			if p.Granted {
				status = SuccessStyle.Render("✓")
			}
			fmt.Fprintf(&b, "%s%s %s\n", prefix, status, style.Render(p.Name))
		}
	}

	if m.permConfirm != "" {
		action := "Grant"
		if m.permConfirm == "revoke" {
			action = "Revoke"
		}
		b.WriteString("\n" + DialogStyle.Render(
			fmt.Sprintf("%s %s? [y/N]", action, m.permConfirmName)) + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "select"),
		keyHint("G", "grant"),
		keyHint("R", "revoke"),
		keyHint("r", "refresh"),
		keyHint("esc", "back"),
	))
	return b.String()
}
