package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	heading      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B48EFA"))
	harnessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B48EFA"))
	ageStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCAA72"))
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E8F0"))
	muted        = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

type picker struct {
	items     []choice
	nameWidth int
	filtered  []int
	query     string
	cursor    int
	offset    int
	width     int
	height    int
	picked    *choice
	now       time.Time
}

func newPicker(items []choice) *picker {
	m := &picker{items: items, nameWidth: 7, width: 80, height: 24, now: time.Now()}
	for _, c := range items {
		if width := lipgloss.Width(c.harness.Name()); width > m.nameWidth {
			m.nameWidth = width
		}
	}
	m.filter()
	return m
}

func (m *picker) Init() tea.Cmd { return nil }

func (m *picker) filter() {
	m.filtered = nil
	for i, c := range m.items {
		label := c.harness.Name()
		if c.session != nil {
			label += " " + c.session.Title
		} else {
			label += " new"
		}
		if strings.Contains(strings.ToLower(label), strings.ToLower(m.query)) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor, m.offset = 0, 0
}

func (m *picker) visibleRows() int {
	n := m.height - 6
	if n < 1 {
		return 1
	}
	return n
}

func (m *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) != 0 {
				c := m.items[m.filtered[m.cursor]]
				m.picked = &c
				return m, tea.Quit
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "pgup":
			m.cursor -= m.visibleRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			m.cursor += m.visibleRows()
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
		case "backspace", "ctrl+h":
			if len(m.query) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.query)
				m.query = m.query[:len(m.query)-size]
				m.filter()
			}
		case "ctrl+u":
			m.query = ""
			m.filter()
		default:
			if text := msg.Key().Text; text != "" {
				m.query += text
				m.filter()
			}
		}
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows() {
		m.offset = m.cursor - m.visibleRows() + 1
	}
	return m, nil
}

func age(t, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func compact(s string, width int) string {
	s = strings.Join(strings.Fields(s), " ")
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}

func renderChoice(c choice, now time.Time, width, nameWidth int, selected bool) string {
	var b strings.Builder
	remaining := width
	appendPart := func(text string, style lipgloss.Style) {
		if remaining <= 0 {
			return
		}
		truncated := lipgloss.Width(text) > remaining
		if truncated {
			text = compact(text, remaining)
		}
		remaining -= lipgloss.Width(text)
		b.WriteString(style.Bold(selected).Render(text))
		if truncated {
			remaining = 0
		}
	}
	if c.session == nil {
		appendPart("+ new ", muted)
		appendPart(c.harness.Name(), harnessStyle)
	} else {
		appendPart(fmt.Sprintf("%-*s", nameWidth, c.harness.Name()), harnessStyle)
		appendPart(fmt.Sprintf(" %4s  ", age(c.session.Modified, now)), ageStyle)
		appendPart(strings.Join(strings.Fields(c.session.Title), " "), titleStyle)
	}
	return b.String()
}

func (m *picker) View() tea.View {
	var b strings.Builder
	b.WriteString(heading.Render("apicker"))
	b.WriteString("  ")
	b.WriteString(muted.Render("start or resume an agent"))
	b.WriteString("\n\n")
	b.WriteString(heading.Render("❯ "))
	b.WriteString(m.query)
	b.WriteString("▏\n\n")
	if len(m.filtered) == 0 {
		b.WriteString(muted.Render("  No matches"))
		b.WriteByte('\n')
	}
	end := m.offset + m.visibleRows()
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := m.offset; i < end; i++ {
		c := m.items[m.filtered[i]]
		selected := i == m.cursor
		prefix := "  "
		if selected {
			prefix = "› "
		}
		b.WriteString(prefix + renderChoice(c, m.now, m.width-4, m.nameWidth, selected) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(muted.Render(fmt.Sprintf("%d matches  ·  ↑/↓ navigate  ·  enter select  ·  esc cancel", len(m.filtered))))
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
