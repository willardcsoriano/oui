package tui

import (
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
// The active provider is read from session.Get() on each turn so that
// /provider and /model switches take effect immediately.
func RunRepl(hooksCfg hooks.Config) error {
	// Save terminal state unconditionally — restored on exit even if BubbleTea
	// crashes without cleaning up.
	if oldState, err := term.GetState(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
	}

	ta := textarea.New()
	ta.Prompt = "> "
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.CharLimit = 0
	// Remove cursor-line background highlight — looks odd in inline mode.
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	// Bold bright-white prompt to match the original textinput design.
	boldWhite := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	ta.FocusedStyle.Prompt = boldWhite
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

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

	prog := tea.NewProgram(m)
	shared.prog = prog

	_, err := prog.Run()
	fmt.Println() // ensure cursor lands on a fresh line after exit
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
	width    int
	cancel   context.CancelFunc // cancels the current agent run; nil when idle
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
		// Quit/cancel shortcuts — always handled first, even mid-run.
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		case tea.KeyEsc:
			if m.running {
				// Cancel in-flight agent; stay in REPL (don't quit).
				if m.cancel != nil {
					m.cancel()
				}
				return m, nil
			}
			// Idle: clear the input.
			m.input.Reset()
			return m, nil
		}

		if m.running {
			return m, nil
		}

		// Plain Enter submits; Alt+Enter inserts a newline (multi-line input).
		if msg.Type == tea.KeyEnter && !msg.Alt {
			text := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if text == "" {
				return m, nil
			}
			if text == "exit" || text == "quit" {
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
	return m, cmd
}

func (m replModel) View() string {
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

func (m replModel) execAgent(ctx context.Context, text string) tea.Cmd {
	prog := m.shared.prog
	hooksCfg := m.hooksCfg
	return func() tea.Msg {
		w := newProgWriter(prog)
		err := agent.Run(ctx, text, session.Get(), hooksCfg, w)
		w.flush()
		return agentDoneMsg{err: err}
	}
}

func (m replModel) execCommand(text string) tea.Cmd {
	prog := m.shared.prog
	return func() tea.Msg {
		w := newProgWriter(prog)
		cont, err := commands.Dispatch(text, w)
		w.flush()
		return cmdDoneMsg{cont: cont, err: err}
	}
}

// ── progWriter ────────────────────────────────────────────────────────────────

// progWriter buffers writes to line boundaries, word-wraps each line to the
// current terminal width, and forwards to tea.Program.Println (goroutine-safe).
// Wrapping is done here because BubbleTea disables terminal auto-wrap in inline
// mode, so long lines would otherwise overflow off-screen.
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
