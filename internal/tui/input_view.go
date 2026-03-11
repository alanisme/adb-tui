package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// inputMessage is implemented by all messages routed to InputModel.
type inputMessage interface{ inputMsg() }

type inputActionMsg struct {
	action string
	err    error
}

type screenSizeMsg struct {
	width, height int
	err           error
}

func (inputActionMsg) inputMsg() {}
func (screenSizeMsg) inputMsg()  {}

type InputMode int

const (
	InputModeButtons InputMode = iota
	InputModeText
	InputModeTap
	InputModeSwipe
	InputModeMonkey
	InputModeIntent
	InputModeDeepLink
	InputModeClipboard
	InputModeGestures
	InputModeLongPress
)

type InputModel struct {
	client         *adb.Client
	serial         string
	width          int
	height         int
	mode           InputMode
	cursor         int
	textInput      textinput.Model
	tapX           textinput.Model
	tapY           textinput.Model
	swipeX1        textinput.Model
	swipeY1        textinput.Model
	swipeX2        textinput.Model
	swipeY2        textinput.Model
	focusIdx       int
	statusMsg      string
	err            error
	recording      bool
	recordCancel   *context.CancelFunc // shared pointer survives value copies
	monkeyInput    textinput.Model
	showMonkey     bool
	intentAction   textinput.Model
	intentData     textinput.Model
	intentPkg      textinput.Model
	intentComp     textinput.Model
	intentCat      textinput.Model
	intentFocusIdx int
	deepLinkInput  textinput.Model
	clipInput      textinput.Model
	gestureCursor  int
	screenW        int
	screenH        int
	longPressX     textinput.Model
	longPressY     textinput.Model
	longPressDur   textinput.Model
	longPressFocus int
}

type virtualButton struct {
	label   string
	keycode string
}

var virtualButtons = []virtualButton{
	{"Home", "KEYCODE_HOME"},
	{"Back", "KEYCODE_BACK"},
	{"Menu", "KEYCODE_MENU"},
	{"Recent", "KEYCODE_APP_SWITCH"},
	{"Power", "KEYCODE_POWER"},
	{"Vol Up", "KEYCODE_VOLUME_UP"},
	{"Vol Down", "KEYCODE_VOLUME_DOWN"},
	{"Vol Mute", "KEYCODE_VOLUME_MUTE"},
	{"Camera", "KEYCODE_CAMERA"},
	{"Enter", "KEYCODE_ENTER"},
	{"Del", "KEYCODE_DEL"},
	{"Tab", "KEYCODE_TAB"},
	{"Escape", "KEYCODE_ESCAPE"},
	{"Wake", "KEYCODE_WAKEUP"},
	{"Sleep", "KEYCODE_SLEEP"},
}

func NewInputModel(client *adb.Client) InputModel {
	ti := textinput.New()
	ti.Placeholder = "text to send..."
	ti.CharLimit = 256

	tx := textinput.New()
	tx.Placeholder = "x"
	tx.CharLimit = 6

	ty := textinput.New()
	ty.Placeholder = "y"
	ty.CharLimit = 6

	sx1 := textinput.New()
	sx1.Placeholder = "x1"
	sx1.CharLimit = 6

	sy1 := textinput.New()
	sy1.Placeholder = "y1"
	sy1.CharLimit = 6

	sx2 := textinput.New()
	sx2.Placeholder = "x2"
	sx2.CharLimit = 6

	sy2 := textinput.New()
	sy2.Placeholder = "y2"
	sy2.CharLimit = 6

	mi := textinput.New()
	mi.Placeholder = "event count (e.g. 500)"
	mi.CharLimit = 10

	ia := textinput.New()
	ia.Placeholder = "action (e.g. android.intent.action.VIEW)"
	ia.CharLimit = 256

	id := textinput.New()
	id.Placeholder = "data (e.g. https://...)"
	id.CharLimit = 512

	ip := textinput.New()
	ip.Placeholder = "package (optional)"
	ip.CharLimit = 256

	ic := textinput.New()
	ic.Placeholder = "component (optional)"
	ic.CharLimit = 256

	icat := textinput.New()
	icat.Placeholder = "category (optional)"
	icat.CharLimit = 256

	dl := textinput.New()
	dl.Placeholder = "deep link URL (e.g. https://...)"
	dl.CharLimit = 512

	ci := textinput.New()
	ci.Placeholder = "text to set on clipboard..."
	ci.CharLimit = 512

	lpx := textinput.New()
	lpx.Placeholder = "x"
	lpx.CharLimit = 6

	lpy := textinput.New()
	lpy.Placeholder = "y"
	lpy.CharLimit = 6

	lpd := textinput.New()
	lpd.Placeholder = "duration ms (e.g. 1000)"
	lpd.CharLimit = 6

	noop := context.CancelFunc(func() {})
	return InputModel{
		client:        client,
		textInput:     ti,
		tapX:          tx,
		tapY:          ty,
		swipeX1:       sx1,
		swipeY1:       sy1,
		swipeX2:       sx2,
		swipeY2:       sy2,
		monkeyInput:   mi,
		intentAction:  ia,
		intentData:    id,
		intentPkg:     ip,
		intentComp:    ic,
		intentCat:     icat,
		deepLinkInput: dl,
		clipInput:     ci,
		longPressX:    lpx,
		longPressY:    lpy,
		longPressDur:  lpd,
		recordCancel:  &noop,
	}
}

func (m InputModel) IsInputCaptured() bool {
	return m.mode != InputModeButtons
}

func (m InputModel) Init() tea.Cmd {
	return nil
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case inputActionMsg:
		if strings.Contains(msg.action, "recording") || strings.Contains(msg.action, "screenrecord") {
			m.recording = false
		}
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else if strings.HasPrefix(msg.action, "clipboard: ") {
			m.statusMsg = SuccessStyle.Render(msg.action)
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action + " sent")
		}
		return m, clearStatusAfter(5 * time.Second)

	case screenSizeMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Screen size: " + msg.err.Error())
			m.mode = InputModeButtons
			return m, clearStatusAfter(5 * time.Second)
		}
		m.screenW = msg.width
		m.screenH = msg.height
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case InputModeButtons:
			return m.updateButtons(msg)
		case InputModeText:
			return m.updateText(msg)
		case InputModeTap:
			return m.updateTap(msg)
		case InputModeSwipe:
			return m.updateSwipe(msg)
		case InputModeMonkey:
			return m.updateMonkey(msg)
		case InputModeIntent:
			return m.updateIntent(msg)
		case InputModeDeepLink:
			return m.updateDeepLink(msg)
		case InputModeClipboard:
			return m.updateClipboard(msg)
		case InputModeGestures:
			return m.updateGestures(msg)
		case InputModeLongPress:
			return m.updateLongPress(msg)
		}
	}
	return m, nil
}

func (m InputModel) updateButtons(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.cursor < len(virtualButtons)-1 {
			m.cursor++
		}
	case key.Matches(msg, DefaultKeyMap.Enter):
		btn := virtualButtons[m.cursor]
		return m, m.sendKeyEvent(btn.keycode)
	case msg.String() == "t":
		m.mode = InputModeText
		m.textInput.Focus()
		return m, textinput.Blink
	case msg.String() == "p":
		m.mode = InputModeTap
		m.focusIdx = 0
		m.tapX.Focus()
		return m, textinput.Blink
	case msg.String() == "w":
		m.mode = InputModeSwipe
		m.focusIdx = 0
		m.swipeX1.Focus()
		return m, textinput.Blink
	case msg.String() == "s":
		return m, m.takeScreenshot()
	case msg.String() == "v":
		if m.recording {
			// Stop recording early
			if m.recordCancel != nil {
				(*m.recordCancel)()
			}
		} else {
			m.recording = true
			return m, m.startScreenRecord()
		}
	case msg.String() == "m":
		m.mode = InputModeMonkey
		m.showMonkey = true
		m.monkeyInput.Focus()
		return m, textinput.Blink
	case msg.String() == "i":
		m.mode = InputModeIntent
		m.intentFocusIdx = 0
		m.intentAction.SetValue("android.intent.action.VIEW")
		m.intentAction.Focus()
		return m, textinput.Blink
	case msg.String() == "l":
		m.mode = InputModeDeepLink
		m.deepLinkInput.Focus()
		return m, textinput.Blink
	case msg.String() == "c":
		return m, m.readClipboard()
	case msg.String() == "C":
		m.mode = InputModeClipboard
		m.clipInput.Focus()
		return m, textinput.Blink
	case msg.String() == "L":
		m.mode = InputModeLongPress
		m.longPressFocus = 0
		m.longPressX.Focus()
		return m, textinput.Blink
	case msg.String() == "g":
		m.mode = InputModeGestures
		m.gestureCursor = 0
		if m.screenW == 0 || m.screenH == 0 {
			return m, m.fetchScreenSize()
		}
	}
	return m, nil
}

func (m InputModel) updateText(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		text := m.textInput.Value()
		m.mode = InputModeButtons
		m.textInput.Reset()
		m.textInput.Blur()
		if text != "" {
			return m, m.sendText(text)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.textInput.Reset()
		m.textInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputModel) updateTap(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.focusIdx = (m.focusIdx + 1) % 2
		if m.focusIdx == 0 {
			m.tapX.Focus()
			m.tapY.Blur()
		} else {
			m.tapX.Blur()
			m.tapY.Focus()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		x := m.tapX.Value()
		y := m.tapY.Value()
		m.mode = InputModeButtons
		m.tapX.Reset()
		m.tapY.Reset()
		m.tapX.Blur()
		m.tapY.Blur()
		if x != "" && y != "" {
			if !isNumeric(x) || !isNumeric(y) {
				m.statusMsg = ErrorStyle.Render("Invalid coordinates: must be numbers")
				return m, nil
			}
			return m, m.sendTap(x, y)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.tapX.Reset()
		m.tapY.Reset()
		m.tapX.Blur()
		m.tapY.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	if m.focusIdx == 0 {
		m.tapX, cmd = m.tapX.Update(msg)
	} else {
		m.tapY, cmd = m.tapY.Update(msg)
	}
	return m, cmd
}

func (m InputModel) updateSwipe(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.focusIdx = (m.focusIdx + 1) % 4
		m.swipeX1.Blur()
		m.swipeY1.Blur()
		m.swipeX2.Blur()
		m.swipeY2.Blur()
		switch m.focusIdx {
		case 0:
			m.swipeX1.Focus()
		case 1:
			m.swipeY1.Focus()
		case 2:
			m.swipeX2.Focus()
		case 3:
			m.swipeY2.Focus()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		x1 := m.swipeX1.Value()
		y1 := m.swipeY1.Value()
		x2 := m.swipeX2.Value()
		y2 := m.swipeY2.Value()
		m.mode = InputModeButtons
		m.swipeX1.Reset()
		m.swipeY1.Reset()
		m.swipeX2.Reset()
		m.swipeY2.Reset()
		m.swipeX1.Blur()
		m.swipeY1.Blur()
		m.swipeX2.Blur()
		m.swipeY2.Blur()
		if x1 != "" && y1 != "" && x2 != "" && y2 != "" {
			if !isNumeric(x1) || !isNumeric(y1) || !isNumeric(x2) || !isNumeric(y2) {
				m.statusMsg = ErrorStyle.Render("Invalid coordinates: must be numbers")
				return m, nil
			}
			return m, m.sendSwipe(x1, y1, x2, y2)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.swipeX1.Reset()
		m.swipeY1.Reset()
		m.swipeX2.Reset()
		m.swipeY2.Reset()
		m.swipeX1.Blur()
		m.swipeY1.Blur()
		m.swipeX2.Blur()
		m.swipeY2.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	switch m.focusIdx {
	case 0:
		m.swipeX1, cmd = m.swipeX1.Update(msg)
	case 1:
		m.swipeY1, cmd = m.swipeY1.Update(msg)
	case 2:
		m.swipeX2, cmd = m.swipeX2.Update(msg)
	case 3:
		m.swipeY2, cmd = m.swipeY2.Update(msg)
	}
	return m, cmd
}

func (m InputModel) updateMonkey(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		countStr := m.monkeyInput.Value()
		m.mode = InputModeButtons
		m.showMonkey = false
		m.monkeyInput.Reset()
		m.monkeyInput.Blur()
		if countStr != "" {
			count, err := strconv.Atoi(countStr)
			if err != nil || count <= 0 {
				m.statusMsg = ErrorStyle.Render("Invalid event count: must be a positive number")
				return m, nil
			}
			return m, m.runMonkey(count)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.showMonkey = false
		m.monkeyInput.Reset()
		m.monkeyInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.monkeyInput, cmd = m.monkeyInput.Update(msg)
	return m, cmd
}

func (m InputModel) updateIntent(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.intentFocusIdx = (m.intentFocusIdx + 1) % 5
		m.intentAction.Blur()
		m.intentData.Blur()
		m.intentPkg.Blur()
		m.intentComp.Blur()
		m.intentCat.Blur()
		switch m.intentFocusIdx {
		case 0:
			m.intentAction.Focus()
		case 1:
			m.intentData.Focus()
		case 2:
			m.intentPkg.Focus()
		case 3:
			m.intentComp.Focus()
		case 4:
			m.intentCat.Focus()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		intent := adb.Intent{
			Action:    m.intentAction.Value(),
			Data:      m.intentData.Value(),
			Component: m.intentComp.Value(),
			Category:  m.intentCat.Value(),
		}
		pkg := m.intentPkg.Value()
		m.mode = InputModeButtons
		m.intentAction.Reset()
		m.intentData.Reset()
		m.intentPkg.Reset()
		m.intentComp.Reset()
		m.intentCat.Reset()
		m.intentAction.Blur()
		m.intentData.Blur()
		m.intentPkg.Blur()
		m.intentComp.Blur()
		m.intentCat.Blur()
		if intent.Action != "" || intent.Data != "" {
			return m, m.sendIntent(intent, pkg)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.intentAction.Reset()
		m.intentData.Reset()
		m.intentPkg.Reset()
		m.intentComp.Reset()
		m.intentCat.Reset()
		m.intentAction.Blur()
		m.intentData.Blur()
		m.intentPkg.Blur()
		m.intentComp.Blur()
		m.intentCat.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	switch m.intentFocusIdx {
	case 0:
		m.intentAction, cmd = m.intentAction.Update(msg)
	case 1:
		m.intentData, cmd = m.intentData.Update(msg)
	case 2:
		m.intentPkg, cmd = m.intentPkg.Update(msg)
	case 3:
		m.intentComp, cmd = m.intentComp.Update(msg)
	case 4:
		m.intentCat, cmd = m.intentCat.Update(msg)
	}
	return m, cmd
}

func (m InputModel) updateDeepLink(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		url := m.deepLinkInput.Value()
		m.mode = InputModeButtons
		m.deepLinkInput.Reset()
		m.deepLinkInput.Blur()
		if url != "" {
			return m, m.sendDeepLink(url)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.deepLinkInput.Reset()
		m.deepLinkInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.deepLinkInput, cmd = m.deepLinkInput.Update(msg)
	return m, cmd
}

func (m InputModel) updateClipboard(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		text := m.clipInput.Value()
		m.mode = InputModeButtons
		m.clipInput.Reset()
		m.clipInput.Blur()
		if text != "" {
			return m, m.writeClipboard(text)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.clipInput.Reset()
		m.clipInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.clipInput, cmd = m.clipInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	var b strings.Builder

	title := HeaderStyle.Render("Input Control")
	b.WriteString(title)
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	switch m.mode {
	case InputModeButtons:
		b.WriteString(m.viewButtons())
	case InputModeText:
		b.WriteString("\n  " + AccentStyle.Render("Send Text") + "\n")
		b.WriteString("  " + m.textInput.View() + "\n")
		b.WriteString("  " + DimStyle.Render("enter: send  esc: cancel") + "\n")
	case InputModeTap:
		b.WriteString("\n  " + AccentStyle.Render("Tap Coordinates") + "\n")
		fmt.Fprintf(&b, "  X: %s  Y: %s\n", m.tapX.View(), m.tapY.View())
		b.WriteString("  " + DimStyle.Render("tab: switch  enter: tap  esc: cancel") + "\n")
	case InputModeSwipe:
		b.WriteString("\n  " + AccentStyle.Render("Swipe Coordinates") + "\n")
		fmt.Fprintf(&b, "  From X: %s  Y: %s\n", m.swipeX1.View(), m.swipeY1.View())
		fmt.Fprintf(&b, "  To   X: %s  Y: %s\n", m.swipeX2.View(), m.swipeY2.View())
		b.WriteString("  " + DimStyle.Render("tab: switch  enter: swipe  esc: cancel") + "\n")
	case InputModeMonkey:
		b.WriteString("\n  " + AccentStyle.Render("Monkey Test") + "\n")
		b.WriteString("  Events: " + m.monkeyInput.View() + "\n")
		b.WriteString("  " + DimStyle.Render("enter: run  esc: cancel") + "\n")
	case InputModeIntent:
		b.WriteString("\n  " + AccentStyle.Render("Send Intent") + "\n")
		b.WriteString("  Action:    " + m.intentAction.View() + "\n")
		b.WriteString("  Data:      " + m.intentData.View() + "\n")
		b.WriteString("  Package:   " + m.intentPkg.View() + "\n")
		b.WriteString("  Component: " + m.intentComp.View() + "\n")
		b.WriteString("  Category:  " + m.intentCat.View() + "\n")
		b.WriteString("  " + DimStyle.Render("tab: next field  enter: send  esc: cancel") + "\n")
	case InputModeDeepLink:
		b.WriteString("\n  " + AccentStyle.Render("Deep Link") + "\n")
		b.WriteString("  URL: " + m.deepLinkInput.View() + "\n")
		b.WriteString("  " + DimStyle.Render("enter: open  esc: cancel") + "\n")
	case InputModeClipboard:
		b.WriteString("\n  " + AccentStyle.Render("Set Clipboard") + "\n")
		b.WriteString("  " + m.clipInput.View() + "\n")
		b.WriteString("  " + DimStyle.Render("enter: set  esc: cancel") + "\n")
	case InputModeGestures:
		b.WriteString(m.viewGestures())
	case InputModeLongPress:
		b.WriteString("\n  " + AccentStyle.Render("Long Press") + "\n")
		fmt.Fprintf(&b, "  X: %s  Y: %s\n", m.longPressX.View(), m.longPressY.View())
		b.WriteString("  Duration: " + m.longPressDur.View() + "\n")
		b.WriteString("  " + DimStyle.Render("tab: switch  enter: press  esc: cancel") + "\n")
	}

	if m.recording {
		b.WriteString("\n  " + WarningStyle.Render("Recording... press v to stop") + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	switch m.mode {
	case InputModeButtons:
		recordHint := keyHint("v", "record")
		if m.recording {
			recordHint = keyHint("v", "stop rec")
		}
		b.WriteString("\n")
		b.WriteString(helpBar(
			keyHint("⏎", "press"),
			keyHint("t", "text"),
			keyHint("p", "tap"),
			keyHint("L", "longpress"),
			keyHint("w", "swipe"),
			keyHint("g", "gestures"),
			keyHint("s", "screenshot"),
			recordHint,
			keyHint("m", "monkey"),
			keyHint("i", "intent"),
			keyHint("l", "deeplink"),
			keyHint("c/C", "clipboard"),
		))
	case InputModeGestures:
		b.WriteString("\n")
		b.WriteString(helpBar(
			keyHint("j/k", "select"),
			keyHint("⏎", "execute"),
			keyHint("esc", "back"),
		))
	}

	return b.String()
}

func (m InputModel) viewButtons() string {
	var b strings.Builder
	b.WriteString("\n  " + AccentStyle.Render("Virtual Keys") + "\n\n")

	cols := 3
	for i, btn := range virtualButtons {
		if i > 0 && i%cols == 0 {
			b.WriteString("\n")
		}
		style := NormalStyle
		prefix := "  "
		if i == m.cursor {
			style = SelectedStyle
			prefix = CursorStyle.Render("▸ ")
		}
		fmt.Fprintf(&b, "%s%-14s", prefix, style.Render(btn.label))
	}
	b.WriteString("\n")
	return b.String()
}

func (m InputModel) viewGestures() string {
	var b strings.Builder
	b.WriteString("\n  " + AccentStyle.Render("Gestures") + "\n\n")

	gestures := adb.Gestures
	cols := 3
	for i, g := range gestures {
		if i > 0 && i%cols == 0 {
			b.WriteString("\n")
		}
		style := NormalStyle
		prefix := "  "
		if i == m.gestureCursor {
			style = SelectedStyle
			prefix = CursorStyle.Render("▸ ")
		}
		fmt.Fprintf(&b, "%s%-14s", prefix, style.Render(g.Name))
	}
	b.WriteString("\n")

	if m.screenW > 0 && m.screenH > 0 {
		b.WriteString("\n  " + DimStyle.Render(fmt.Sprintf("Screen: %d×%d", m.screenW, m.screenH)) + "\n")
	} else {
		b.WriteString("\n  " + DimStyle.Render("Loading screen size...") + "\n")
	}

	return b.String()
}

func (m InputModel) SetDevice(serial string) InputModel {
	m.serial = serial
	m.screenW = 0
	m.screenH = 0
	return m
}

func (m InputModel) SetSize(w, h int) InputModel {
	m.width = w
	m.height = h
	return m
}

func (m InputModel) sendKeyEvent(keycode string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ShellArgs(ctx, serial, "input", "keyevent", keycode)
		return inputActionMsg{action: keycode, err: err}
	}
}

func (m InputModel) sendText(text string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// adb shell input text requires spaces as %s and special chars escaped
		escaped := strings.ReplaceAll(text, " ", "%s")
		_, err := client.ShellArgs(ctx, serial, "input", "text", escaped)
		return inputActionMsg{action: "text", err: err}
	}
}

func (m InputModel) sendTap(x, y string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ShellArgs(ctx, serial, "input", "tap", x, y)
		return inputActionMsg{action: fmt.Sprintf("tap(%s,%s)", x, y), err: err}
	}
}

func (m InputModel) sendSwipe(x1, y1, x2, y2 string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ShellArgs(ctx, serial, "input", "swipe", x1, y1, x2, y2)
		return inputActionMsg{action: "swipe", err: err}
	}
}

func (m InputModel) updateGestures(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	gestures := adb.Gestures
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.gestureCursor > 0 {
			m.gestureCursor--
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.gestureCursor < len(gestures)-1 {
			m.gestureCursor++
		}
	case key.Matches(msg, DefaultKeyMap.Enter):
		if m.screenW == 0 || m.screenH == 0 {
			m.statusMsg = ErrorStyle.Render("Screen size not available yet")
			return m, nil
		}
		g := gestures[m.gestureCursor]
		return m, m.sendGesture(g)
	case msg.Type == tea.KeyEsc:
		m.mode = InputModeButtons
		return m, nil
	}
	return m, nil
}

func (m InputModel) sendGesture(g adb.GestureDef) tea.Cmd {
	serial := m.serial
	client := m.client
	w, h := m.screenW, m.screenH
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.HumanSwipe(ctx, serial, w, h, g.Params)
		return inputActionMsg{action: g.Name, err: err}
	}
}

func (m InputModel) fetchScreenSize() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		w, h, err := client.GetScreenSize(ctx, serial)
		return screenSizeMsg{width: w, height: h, err: err}
	}
}

func (m InputModel) updateLongPress(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.longPressFocus = (m.longPressFocus + 1) % 3
		m.longPressX.Blur()
		m.longPressY.Blur()
		m.longPressDur.Blur()
		switch m.longPressFocus {
		case 0:
			m.longPressX.Focus()
		case 1:
			m.longPressY.Focus()
		case 2:
			m.longPressDur.Focus()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		x := m.longPressX.Value()
		y := m.longPressY.Value()
		dur := m.longPressDur.Value()
		m.mode = InputModeButtons
		m.longPressX.Reset()
		m.longPressY.Reset()
		m.longPressDur.Reset()
		m.longPressX.Blur()
		m.longPressY.Blur()
		m.longPressDur.Blur()
		if x != "" && y != "" {
			if !isNumeric(x) || !isNumeric(y) {
				m.statusMsg = ErrorStyle.Render("Invalid coordinates: must be numbers")
				return m, nil
			}
			durationMs := 1000
			if dur != "" {
				if !isNumeric(dur) {
					m.statusMsg = ErrorStyle.Render("Invalid duration: must be a number")
					return m, nil
				}
				durationMs, _ = strconv.Atoi(dur)
				if durationMs <= 0 {
					durationMs = 1000
				}
			}
			xi, _ := strconv.Atoi(x)
			yi, _ := strconv.Atoi(y)
			return m, m.sendLongPress(xi, yi, durationMs)
		}
		return m, nil
	case tea.KeyEsc:
		m.mode = InputModeButtons
		m.longPressX.Reset()
		m.longPressY.Reset()
		m.longPressDur.Reset()
		m.longPressX.Blur()
		m.longPressY.Blur()
		m.longPressDur.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	switch m.longPressFocus {
	case 0:
		m.longPressX, cmd = m.longPressX.Update(msg)
	case 1:
		m.longPressY, cmd = m.longPressY.Update(msg)
	case 2:
		m.longPressDur, cmd = m.longPressDur.Update(msg)
	}
	return m, cmd
}

func (m InputModel) sendLongPress(x, y, durationMs int) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(durationMs+5000)*time.Millisecond)
		defer cancel()
		err := client.LongPress(ctx, serial, x, y, durationMs)
		return inputActionMsg{action: fmt.Sprintf("longpress(%d,%d,%dms)", x, y, durationMs), err: err}
	}
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func (m InputModel) takeScreenshot() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ts := time.Now().Format("20060102_150405")
		localPath := filepath.Join(downloadsDir(), fmt.Sprintf("screenshot_%s.png", ts))
		err := client.Screenshot(ctx, serial, localPath)
		if err != nil {
			return inputActionMsg{action: "screenshot", err: err}
		}
		return inputActionMsg{action: fmt.Sprintf("screenshot saved to %s", localPath)}
	}
}

func (m InputModel) runMonkey(count int) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		cmd := fmt.Sprintf("monkey -v %d", count)
		_, err := client.Shell(ctx, serial, cmd)
		return inputActionMsg{action: fmt.Sprintf("monkey %d events", count), err: err}
	}
}

func (m InputModel) sendIntent(intent adb.Intent, pkg string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		args := []string{"am", "start"}
		if pkg != "" {
			args = append(args, "-p", pkg)
		}
		args = append(args, intent.Args()...)
		_, err := client.ShellArgs(ctx, serial, args...)
		return inputActionMsg{action: "intent", err: err}
	}
}

func (m InputModel) sendDeepLink(url string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.StartActivity(ctx, serial, adb.Intent{
			Action: "android.intent.action.VIEW",
			Data:   url,
		})
		return inputActionMsg{action: "deeplink", err: err}
	}
}

func (m InputModel) readClipboard() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		text, err := client.GetClipboard(ctx, serial)
		if err != nil {
			return inputActionMsg{action: "clipboard read", err: err}
		}
		return inputActionMsg{action: fmt.Sprintf("clipboard: %s", text)}
	}
}

func (m InputModel) writeClipboard(text string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetClipboard(ctx, serial, text)
		return inputActionMsg{action: "clipboard set", err: err}
	}
}

func (m InputModel) startScreenRecord() tea.Cmd {
	serial := m.serial
	client := m.client

	// Stop function: send SIGINT to the remote screenrecord process so it
	// can finalize the MP4 (write moov atom). Killing the local adb process
	// instead would leave the file corrupt and unplayable.
	*m.recordCancel = func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_, _ = client.ShellArgs(stopCtx, serial, "pkill", "-INT", "screenrecord")
	}

	return func() tea.Msg {
		ts := time.Now().Format("20060102_150405")
		remotePath := fmt.Sprintf("/sdcard/record_%s.mp4", ts)
		localPath := filepath.Join(downloadsDir(), fmt.Sprintf("record_%s.mp4", ts))

		ctx := context.Background()
		cmd, err := client.ScreenRecord(ctx, serial, remotePath, adb.ScreenRecordOptions{TimeLimit: 30})
		if err != nil {
			return inputActionMsg{action: "screenrecord", err: err}
		}
		// Wait blocks until screenrecord ends (--time-limit or remote SIGINT from user pressing v)
		_ = cmd.Wait()

		// Small delay to let screenrecord flush the file on device
		time.Sleep(500 * time.Millisecond)

		// Pull the recorded file
		pullCtx, pullCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer pullCancel()
		pullErr := client.Pull(pullCtx, serial, remotePath, localPath)
		if pullErr != nil {
			return inputActionMsg{action: "screenrecord", err: pullErr}
		}

		// Clean up remote file
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		_, _ = client.ShellArgs(cleanCtx, serial, "rm", remotePath)

		return inputActionMsg{action: fmt.Sprintf("recording saved to %s", localPath)}
	}
}
