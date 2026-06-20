package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 4096

	shared := &sharedProg{}
	m := replModel{
		input:    ti,
		shared:   shared,
		hooksCfg: hooksCfg,
	}

	// NewProgram does NOT start the event loop — Run() does.
	// Setting shared.prog here is safe before any events are processed.
	prog := tea.NewProgram(m, tea.WithoutSignalHandler())
	shared.prog = prog

	_, err := prog.Run()
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
	input    textinput.Model
	running  bool
	shared   *sharedProg
	hooksCfg hooks.Config
}

func (m replModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m replModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.running {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.shared.prog.Println("bye")
			return m, tea.Quit

		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if text == "" {
				return m, nil
			}
			if text == "exit" || text == "quit" {
				m.shared.prog.Println("bye")
				return m, tea.Quit
			}
			m.running = true
			if strings.HasPrefix(text, "/") {
				return m, m.execCommand(text)
			}
			return m, m.execAgent(text)
		}

	case agentDoneMsg:
		m.running = false
		if msg.err != nil {
			m.shared.prog.Println("\033[31merror: " + msg.err.Error() + "\033[0m")
		}
		return m, textinput.Blink

	case cmdDoneMsg:
		m.running = false
		if msg.err != nil {
			m.shared.prog.Println("\033[31merror: " + msg.err.Error() + "\033[0m")
		}
		if !msg.cont {
			return m, tea.Quit
		}
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m replModel) View() string {
	if m.running {
		return ""
	}
	return "\033[1;36m>\033[0m " + m.input.View()
}

func (m replModel) execAgent(text string) tea.Cmd {
	prog := m.shared.prog
	hooksCfg := m.hooksCfg
	return func() tea.Msg {
		w := newProgWriter(prog)
		err := agent.Run(context.Background(), text, session.Get(), hooksCfg, w)
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

// progWriter buffers writes to line boundaries and forwards each complete
// line to tea.Program.Println (goroutine-safe).
type progWriter struct {
	prog *tea.Program
	mu   sync.Mutex
	buf  strings.Builder
}

func newProgWriter(prog *tea.Program) *progWriter {
	return &progWriter{prog: prog}
}

func (w *progWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, b := range p {
		if b == '\n' {
			w.prog.Println(w.buf.String())
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
		w.prog.Println(w.buf.String())
		w.buf.Reset()
	}
}

var _ io.Writer = (*progWriter)(nil)
