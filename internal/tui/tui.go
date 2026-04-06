package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	chatStyle = lipgloss.NewStyle()
	inputBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Padding(0, 1)
	userStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	agentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	metaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Run starts the chat TUI until the user quits. If raw is false, agent replies are
// rendered as terminal markdown (via Glamour). If raw is true, reply text is shown unchanged.
// If debug is true, each turn logs indented JSON for the SendMessage request and response
// before the usual transcript lines.
func Run(ctx context.Context, client *a2aclient.Client, agentName string, raw, debug bool) error {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Message — Enter to send · Esc quit · wheel to scroll"
	ti.Focus()
	ti.CharLimit = 16000

	vp := viewport.New(80, 12)

	m := &model{
		client:     client,
		agentName:  agentName,
		raw:        raw,
		debug:      debug,
		input:      ti,
		viewport:   vp,
		ctx:        ctx,
		termWidth:  80,
		termHeight: 24,
	}
	m.ensureMarkdownRenderer()
	m.refreshChat()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

type model struct {
	client    *a2aclient.Client
	agentName string
	input     textinput.Model
	viewport  viewport.Model
	ctx       context.Context

	termWidth  int
	termHeight int

	raw   bool
	debug bool

	glamour         *glamour.TermRenderer
	glamourWordWrap int

	busy         bool
	spinnerPhase int
	log          strings.Builder
}

type sendCompleteMsg struct {
	err     error
	reply   string
	reqJSON string
	resJSON string
}

type spinnerTickMsg struct{}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyLayout(msg.Width, msg.Height)
		m.refreshChat()
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case spinnerTickMsg:
		if !m.busy {
			return m, nil
		}
		m.spinnerPhase++
		m.refreshChat()
		return m, spinnerTick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.busy {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.appendLine(userStyle.Render("You: ") + text)
			m.input.SetValue("")
			m.busy = true
			m.spinnerPhase = 0
			m.refreshChat()
			return m, tea.Batch(m.sendCmd(text), spinnerTick())
		}

	case sendCompleteMsg:
		m.busy = false
		if m.debug && msg.reqJSON != "" {
			m.appendLine(metaStyle.Render("── SendMessage request ──") + "\n" + msg.reqJSON)
		}
		if msg.err != nil {
			m.appendLine(errStyle.Render("Error: ") + formatChatError(msg.err))
		} else {
			if m.debug && msg.resJSON != "" {
				m.appendLine(metaStyle.Render("── SendMessage response ──") + "\n" + msg.resJSON)
			}
			body := m.formatAgentReply(msg.reply)
			m.appendLine(agentStyle.Render("Agent:") + "\n" + body)
		}
		m.refreshChat()
		return m, nil
	}

	var cmd tea.Cmd
	if !m.busy {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func spinnerTick() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// refreshChat sets viewport content from the log plus an in-chat loading line while busy.
func (m *model) refreshChat() {
	m.viewport.SetContent(m.chatContent())
	m.viewport.GotoBottom()
}

func (m *model) chatContent() string {
	if m.log.Len() == 0 && !m.busy {
		return metaStyle.Render("Ready. Type a message below.")
	}
	var b strings.Builder
	b.WriteString(m.log.String())
	if m.busy {
		if m.log.Len() > 0 {
			b.WriteString("\n")
		}
		fr := spinnerFrames[m.spinnerPhase%len(spinnerFrames)]
		b.WriteString(agentStyle.Render("Agent: "))
		b.WriteString(metaStyle.Render(fr + " "))
		b.WriteString(metaStyle.Render("waiting…"))
	}
	return b.String()
}

func (m *model) applyLayout(width, height int) {
	if width < 20 {
		width = 20
	}
	if height < 5 {
		height = 5
	}
	m.termWidth = width
	m.termHeight = height

	const headerLines = 1
	const inputLines = 1
	chatH := height - headerLines - inputLines
	if chatH < 3 {
		chatH = 3
	}

	m.viewport.Width = width
	m.viewport.Height = chatH
	m.input.Width = max(8, width-4)
	m.ensureMarkdownRenderer()
}

func (m *model) View() string {
	w := m.termWidth
	if w < 20 {
		w = 80
	}

	header := m.renderHeader(w)
	chat := chatStyle.Width(w).Render(m.viewport.View())
	inputRow := inputBarStyle.Width(w).Render(m.input.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, chat, inputRow)
}

func (m *model) renderHeader(w int) string {
	meta := "m2a"
	if m.busy {
		meta = "sending…"
	}
	pad := 2
	innerW := w - pad
	if innerW < 8 {
		innerW = 8
	}
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(meta)
	rightW := lipgloss.Width(right)
	nameMax := innerW - rightW - 1
	if nameMax < 1 {
		nameMax = 1
	}
	name := truncateRunes(m.agentName, nameMax)
	left := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(name)
	gap := innerW - lipgloss.Width(left) - rightW
	if gap < 1 {
		gap = 1
	}
	row := lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", gap), right)
	return lipgloss.NewStyle().
		Width(w).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Render(row)
}

func truncateRunes(s string, max int) string {
	if max < 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

func formatChatError(err error) string {
	s := err.Error()
	if errors.Is(err, a2a.ErrMethodNotFound) {
		s += "\n→ The server rejected JSON-RPC method SendMessage. Use a full agent card with supportedInterfaces, or point -base at the URL where your stack exposes A2A (many frameworks use HTTP+JSON on the same host)."
	}
	return s
}

func (m *model) appendLine(s string) {
	if m.log.Len() > 0 {
		m.log.WriteString("\n")
	}
	m.log.WriteString(s)
}

func (m *model) ensureMarkdownRenderer() {
	if m.raw {
		m.glamour = nil
		m.glamourWordWrap = 0
		return
	}
	wrap := max(20, m.termWidth-4)
	if m.glamour != nil && m.glamourWordWrap == wrap {
		return
	}
	r, err := newMarkdownRenderer(wrap)
	if err != nil {
		m.glamour = nil
		return
	}
	m.glamour = r
	m.glamourWordWrap = wrap
}

func (m *model) formatAgentReply(text string) string {
	if m.raw {
		return text
	}
	m.ensureMarkdownRenderer()
	return renderMarkdown(m.glamour, text)
}

func (m *model) sendCmd(text string) tea.Cmd {
	return func() tea.Msg {
		ctx := m.ctx
		msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(text))
		req := &a2a.SendMessageRequest{Message: msg}
		var reqJSON string
		if m.debug {
			reqJSON = marshalDebugJSON(req)
		}
		res, err := m.client.SendMessage(ctx, req)
		if err != nil {
			return sendCompleteMsg{err: err, reqJSON: reqJSON}
		}
		out := sendCompleteMsg{reply: formatSendResult(res), reqJSON: reqJSON}
		if m.debug {
			out.resJSON = formatDebugSendMessageResult(res)
		}
		return out
	}
}

// formatSendResult turns a single SendMessage reply into text for the transcript.
// For *Task responses it may include status, server-reported History entries, and artifacts.
func formatSendResult(res a2a.SendMessageResult) string {
	switch v := res.(type) {
	case *a2a.Message:
		return formatParts(v.Parts)
	case *a2a.Task:
		var b strings.Builder
		b.WriteString(fmt.Sprintf("[%s]", v.Status.State))
		if v.Status.Message != nil {
			if t := strings.TrimSpace(messageText(v.Status.Message)); t != "" {
				b.WriteString(" ")
				b.WriteString(t)
			}
		}
		for _, h := range v.History {
			if h.Role == a2a.MessageRoleAgent {
				if t := strings.TrimSpace(messageText(h)); t != "" {
					if b.Len() > 0 {
						b.WriteString("\n")
					}
					b.WriteString(t)
				}
			}
		}
		for _, a := range v.Artifacts {
			if t := strings.TrimSpace(formatParts(a.Parts)); t != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				if a.Name != "" {
					b.WriteString(a.Name + ": ")
				}
				b.WriteString(t)
			}
		}
		out := strings.TrimSpace(b.String())
		if out == "" {
			out = "(empty task response)"
		}
		return out
	default:
		return fmt.Sprintf("%T: %+v", res, res)
	}
}

func messageText(m *a2a.Message) string {
	return formatParts(m.Parts)
}

func formatParts(parts a2a.ContentParts) string {
	if len(parts) == 0 {
		return ""
	}
	var ss []string
	for _, p := range parts {
		if p == nil {
			continue
		}
		if t := strings.TrimSpace(p.Text()); t != "" {
			ss = append(ss, t)
		} else if raw := p.Raw(); len(raw) > 0 {
			mt := p.MediaType
			if mt == "" {
				mt = "application/octet-stream"
			}
			ss = append(ss, fmt.Sprintf("[%s, %d bytes]", mt, len(raw)))
		} else {
			mt := p.MediaType
			if mt == "" {
				mt = "part"
			}
			ss = append(ss, fmt.Sprintf("[%s]", mt))
		}
	}
	return strings.Join(ss, "\n")
}
