package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/term"
	"github.com/willardcsoriano/oui/internal/agent"
	"github.com/willardcsoriano/oui/internal/commands"
	"github.com/willardcsoriano/oui/internal/hooks"
	"github.com/willardcsoriano/oui/internal/session"
)

// sharedProg holds the *tea.Program reference so tea.Cmd goroutines
// can call prog.Println without circular initialization.
type sharedProg struct {
	prog *tea.Program
}

// RunRepl starts the interactive REPL and blocks until the user exits.
func RunRepl(hooksCfg hooks.Config) error {
	if oldState, err := term.GetState(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
	}

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.CharLimit = 0
	// Remove cursor-line highlight — looks odd in inline mode.
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	// Bold bright-white ">" on line 0; plain indent on continuation lines.
	boldWhite := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	ta.FocusedStyle.Prompt = boldWhite
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	ta.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return "> "
		}
		return "  "
	})
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		ta.SetWidth(w)
	} else {
		ta.SetWidth(80)
	}
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	shared := &sharedProg{}
	m := replModel{
		input:    ta,
		spinner:  sp,
		shared:   shared,
		hooksCfg: hooksCfg,
	}

	// shiftEnterReader translates Shift+Enter sequences to 0x1C (Ctrl+\)
	// so BubbleTea surfaces them as KeyCtrlBackslash — a key we can act on.
	stdinReader := &shiftEnterReader{inner: os.Stdin}
	prog := tea.NewProgram(m, tea.WithInput(stdinReader))
	shared.prog = prog

	_, err := prog.Run()
	fmt.Println()
	if err != nil {
		return fmt.Errorf("repl: %w", err)
	}
	return nil
}

// ── Message types ─────────────────────────────────────────────────────────────

type agentDoneMsg struct{ err error }
type cmdDoneMsg struct {
	cont bool
	err  error
}

// ── Model ─────────────────────────────────────────────────────────────────────

type replModel struct {
	input    textarea.Model
	spinner  spinner.Model
	running  bool
	quitting bool
	width    int
	cancel   context.CancelFunc
	shared   *sharedProg
	hooksCfg hooks.Config
}

func (m replModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m replModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.SetWidth(msg.Width)
		return m, nil

	case spinner.TickMsg:
		if m.running {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Quit/cancel — always first, even mid-run.
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			if m.cancel != nil {
				m.cancel()
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEsc:
			if m.running {
				if m.cancel != nil {
					m.cancel()
				}
				return m, nil
			}
			m.input.Reset()
			m.input.SetHeight(1)
			return m, nil

		case tea.KeyCtrlBackslash:
			// Shift+Enter translated by shiftEnterReader — insert newline.
			if !m.running {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m.growInputHeight()
				return m, cmd
			}
			return m, nil
		}

		if m.running {
			return m, nil
		}

		// Alt+Enter = newline (fallback for terminals that don't support
		// modifyOtherKeys or Kitty protocol).
		if msg.Type == tea.KeyEnter && msg.Alt {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.growInputHeight()
			return m, cmd
		}

		// Plain Enter = submit.
		if msg.Type == tea.KeyEnter && !msg.Alt {
			text := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			m.input.SetHeight(1)
			if text == "" {
				return m, nil
			}
			if text == "exit" || text == "quit" {
				m.quitting = true
				return m, tea.Quit
			}
			m.running = true
			if strings.HasPrefix(text, "/") {
				return m, tea.Batch(m.execCommand(text), m.spinner.Tick)
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			return m, tea.Batch(m.execAgent(ctx, text), m.spinner.Tick)
		}

	case agentDoneMsg:
		m.running = false
		m.cancel = nil
		if msg.err != nil && msg.err != context.Canceled {
			m.shared.prog.Println("\033[31merror: " + msg.err.Error() + "\033[0m")
		}
		m.input.Focus()
		return m, textarea.Blink

	case cmdDoneMsg:
		m.running = false
		m.cancel = nil
		if msg.err != nil {
			m.shared.prog.Println("\033[31merror: " + msg.err.Error() + "\033[0m")
		}
		if !msg.cont {
			return m, tea.Quit
		}
		m.input.Focus()
		return m, textarea.Blink
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.growInputHeight()
	return m, cmd
}

// growInputHeight auto-expands the textarea up to 4 lines as content grows.
func (m *replModel) growInputHeight() {
	lines := strings.Count(m.input.Value(), "\n") + 1
	if lines < 1 {
		lines = 1
	}
	if lines > 4 {
		lines = 4
	}
	m.input.SetHeight(lines)
}

func (m replModel) View() string {
	if m.quitting {
		return ""
	}
	w := m.width
	if w == 0 {
		w = 80
	}
	line := "\033[2m" + strings.Repeat("─", w) + "\033[0m"

	if m.running {
		return line + "\n" + m.spinner.View() + " thinking…\n" + line
	}
	return line + "\n" + m.input.View() + "\n" + line
}

var echoStyle = lipgloss.NewStyle().
	Background(lipgloss.AdaptiveColor{Light: "254", Dark: "237"}).
	Bold(true)

func echoPrompt(prog *tea.Program, text string) {
	w := termWidth()
	style := echoStyle.Width(w)
	lines := strings.Split(text, "\n")
	prog.Println(style.Render("> " + lines[0]))
	for _, l := range lines[1:] {
		prog.Println(style.Render("  " + l))
	}
}

func (m replModel) execAgent(ctx context.Context, text string) tea.Cmd {
	prog := m.shared.prog
	hooksCfg := m.hooksCfg
	return func() tea.Msg {
		echoPrompt(prog, text)
		prog.Println("")
		w := newProgWriter(prog)
		err := agent.Run(ctx, text, session.Get(), hooksCfg, w)
		w.flush()
		prog.Println("")
		return agentDoneMsg{err: err}
	}
}

func (m replModel) execCommand(text string) tea.Cmd {
	prog := m.shared.prog
	return func() tea.Msg {
		echoPrompt(prog, text)
		w := newProgWriter(prog)
		cont, err := commands.Dispatch(text, w)
		w.flush()
		return cmdDoneMsg{cont: cont, err: err}
	}
}

// ── shiftEnterReader ──────────────────────────────────────────────────────────

// shiftEnterReader wraps stdin to:
//  1. Send \033[>4;2m (xterm modifyOtherKeys level 2) on the FIRST Read — at
//     that point BubbleTea has already put the terminal in raw mode, so there
//     is no echo and no terminal response bytes queued on stdin.
//  2. Translate \033[27;2;13~ (Shift+Enter in modifyOtherKeys mode) to 0x1C
//     (Ctrl+\) which BubbleTea maps to KeyCtrlBackslash.
//  3. Restore the mode in Close(), which BubbleTea calls on exit, without
//     actually closing os.Stdin (that would break the parent shell).
var (
	seqModifyOtherKeys = []byte("\x1b[27;2;13~") // XTerm modifyOtherKeys level 2
	seqKittyShiftEnter = []byte("\x1b[13;2u")    // Kitty keyboard protocol
	seqSubstitute      = []byte{0x1c}
)

type shiftEnterReader struct {
	inner    *os.File
	buf      []byte
	initOnce sync.Once
}

// Fd, Write, and Close satisfy charmbracelet/x/term.File (io.ReadWriteCloser +
// Fd) so BubbleTea detects this as a terminal and puts stdin into raw mode.
func (r *shiftEnterReader) Fd() uintptr                { return r.inner.Fd() }
func (r *shiftEnterReader) Write(p []byte) (int, error) { return r.inner.Write(p) }

// Close restores modifyOtherKeys but does NOT close os.Stdin.
func (r *shiftEnterReader) Close() error {
	fmt.Fprint(os.Stdout, "\033[>4;0m") //nolint:errcheck
	return nil
}

func (r *shiftEnterReader) Read(p []byte) (int, error) {
	// Enable modifyOtherKeys exactly once, after BubbleTea has set up raw mode.
	// Sending this in cooked mode would cause some terminals to queue response
	// bytes on stdin, which BubbleTea would then misinterpret as user input.
	r.initOnce.Do(func() {
		fmt.Fprint(os.Stdout, "\033[>4;2m") //nolint:errcheck
	})

	if len(r.buf) > 0 {
		n := copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}
	tmp := make([]byte, 256)
	n, err := r.inner.Read(tmp)
	if n == 0 {
		return 0, err
	}
	data := tmp[:n]
	data = bytes.ReplaceAll(data, seqModifyOtherKeys, seqSubstitute)
	data = bytes.ReplaceAll(data, seqKittyShiftEnter, seqSubstitute)
	n2 := copy(p, data)
	if n2 < len(data) {
		r.buf = append(r.buf[:0], data[n2:]...)
	}
	return n2, err
}

// ── progWriter ────────────────────────────────────────────────────────────────

type progWriter struct {
	prog *tea.Program
	mu   sync.Mutex
	buf  strings.Builder
}

func newProgWriter(prog *tea.Program) *progWriter {
	return &progWriter{prog: prog}
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func (w *progWriter) println(line string) {
	width := termWidth()
	wrapped := wordwrap.String(line, width)
	for _, l := range strings.Split(strings.TrimRight(wrapped, "\n"), "\n") {
		w.prog.Println(l)
	}
}

func (w *progWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, b := range p {
		if b == '\n' {
			w.println(w.buf.String())
			w.buf.Reset()
		} else {
			w.buf.WriteByte(b)
		}
	}
	return len(p), nil
}

func (w *progWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() > 0 {
		w.println(w.buf.String())
		w.buf.Reset()
	}
}

var _ io.Writer = (*progWriter)(nil)
