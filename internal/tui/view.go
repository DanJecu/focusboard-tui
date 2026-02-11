package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "25", Dark: "212"}).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "252"})

	completedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "97", Dark: "141"})

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "230", Dark: "230"}).
			Background(lipgloss.AdaptiveColor{Light: "25", Dark: "61"}).
			Bold(true).
			Padding(0, 1)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "243", Dark: "241"})

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "252"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "196"}).
			Bold(true)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "25", Dark: "212"}).
			Bold(true)

	focusedBorderColor = lipgloss.AdaptiveColor{Light: "25", Dark: "212"}
	dimBorderColor     = lipgloss.AdaptiveColor{Light: "243", Dark: "241"}
)

func renderPane(totalWidth, innerHeight int, title string, focused bool, content string) string {
	borderColor := dimBorderColor
	if focused {
		borderColor = focusedBorderColor
	}
	bc := lipgloss.NewStyle().Foreground(borderColor)

	border := lipgloss.RoundedBorder()
	innerWidth := totalWidth - 2

	titleStyle := lipgloss.NewStyle().Foreground(borderColor)
	if focused {
		titleStyle = titleStyle.Bold(true)
	}
	titleStr := titleStyle.Render(" " + title + " ")
	titleVisualWidth := lipgloss.Width(titleStr)

	remainFill := innerWidth - 1 - titleVisualWidth
	if remainFill < 0 {
		remainFill = 0
	}
	topLine := bc.Render(border.TopLeft) +
		bc.Render(border.Top) +
		titleStr +
		bc.Render(strings.Repeat(border.Top, remainFill)) +
		bc.Render(border.TopRight)
	if titleVisualWidth+1 >= innerWidth {
		topLine = bc.Render(border.TopLeft) + titleStr + bc.Render(border.TopRight)
	}

	paddedWidth := innerWidth - 2
	if paddedWidth < 0 {
		paddedWidth = 0
	}
	bodyStyle := lipgloss.NewStyle().Width(paddedWidth).Height(innerHeight)
	body := bodyStyle.Render(content)

	bodyLines := strings.Split(body, "\n")
	var middle strings.Builder
	for _, line := range bodyLines {
		middle.WriteString(bc.Render(border.Left) + " " + line + " " + bc.Render(border.Right) + "\n")
	}

	bottomLine := bc.Render(border.BottomLeft) +
		bc.Render(strings.Repeat(border.Bottom, innerWidth)) +
		bc.Render(border.BottomRight)

	return topLine + "\n" + middle.String() + bottomLine
}

func helpKey(key, desc string) string {
	return keyStyle.Render(key) + " " + descStyle.Render(desc)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	bottomLines := 2
	panelH := m.height - bottomLines - 2
	if panelH < 1 {
		panelH = 1
	}

	leftTotal := m.width / 3
	rightTotal := m.width - leftTotal

	projectTitle := "Projects"
	leftFocused := m.focus == focusProjects && m.mode == modeNormal
	leftContent := m.padContent(m.projectLines(), panelH)
	leftPane := renderPane(leftTotal, panelH, projectTitle, leftFocused, leftContent)

	todoTitle := "Todos"
	if p := m.currentProject(); p != nil {
		todoTitle = fmt.Sprintf("Todos: %s", p.Name)
	}
	rightFocused := m.focus == focusTodos && m.mode == modeNormal
	rightContent := m.padContent(m.todoLines(), panelH)
	rightPane := renderPane(rightTotal, panelH, todoTitle, rightFocused, rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	var b strings.Builder
	b.WriteString(panels)
	b.WriteString("\n")

	help := strings.Join([]string{
		helpKey("j/k", "nav"),
		helpKey("ctrl+h/l", "switch"),
		helpKey("enter", "toggle"),
		helpKey("a", "add"),
		helpKey("e", "edit"),
		helpKey("d", "del"),
		helpKey("l", "link"),
		helpKey("o", "open"),
		helpKey("q", "quit"),
	}, "  ")
	b.WriteString(help)
	b.WriteString("\n")

	if m.statusErr {
		b.WriteString(errorStyle.Render("ERROR: " + m.status))
	} else {
		b.WriteString(statusStyle.Render(m.status))
	}

	baseView := b.String()

	if m.mode == modeInput || m.mode == modeConfirmDelete {
		popupWidth := m.width / 3
		if popupWidth < 40 {
			popupWidth = 40
		}
		if popupWidth > m.width-4 {
			popupWidth = m.width - 4
		}

		var title, body string
		if m.mode == modeInput {
			title = m.inputTitle()
			m.input.SetWidth(popupWidth - 4)
			body = m.input.View()
		} else {
			title = "Confirm"
			body = m.deleteMessage
		}

		popup := renderPopup(popupWidth, title, body)
		return overlayCenter(baseView, popup, m.width, m.height)
	}

	return baseView
}

func renderPopup(width int, title, body string) string {
	bc := lipgloss.NewStyle().Foreground(focusedBorderColor)
	border := lipgloss.RoundedBorder()
	innerWidth := width - 2

	titleStyle := lipgloss.NewStyle().Foreground(focusedBorderColor).Bold(true)
	titleStr := titleStyle.Render(" " + title + " ")
	titleVisualWidth := lipgloss.Width(titleStr)

	remainFill := innerWidth - 1 - titleVisualWidth
	if remainFill < 0 {
		remainFill = 0
	}
	topLine := bc.Render(border.TopLeft) +
		bc.Render(border.Top) +
		titleStr +
		bc.Render(strings.Repeat(border.Top, remainFill)) +
		bc.Render(border.TopRight)

	paddedWidth := innerWidth - 2
	if paddedWidth < 0 {
		paddedWidth = 0
	}
	bodyStyle := lipgloss.NewStyle().Width(paddedWidth)
	renderedBody := bodyStyle.Render(body)

	var middle strings.Builder
	middle.WriteString(bc.Render(border.Left) + strings.Repeat(" ", innerWidth) + bc.Render(border.Right) + "\n")
	for _, line := range strings.Split(renderedBody, "\n") {
		pad := innerWidth - lipgloss.Width(line) - 2
		if pad < 0 {
			pad = 0
		}
		middle.WriteString(bc.Render(border.Left) + " " + line + strings.Repeat(" ", pad) + " " + bc.Render(border.Right) + "\n")
	}
	middle.WriteString(bc.Render(border.Left) + strings.Repeat(" ", innerWidth) + bc.Render(border.Right) + "\n")

	bottomLine := bc.Render(border.BottomLeft) +
		bc.Render(strings.Repeat(border.Bottom, innerWidth)) +
		bc.Render(border.BottomRight)

	return topLine + "\n" + middle.String() + bottomLine
}

func overlayCenter(bg, fg string, bgWidth, bgHeight int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for len(bgLines) < bgHeight {
		bgLines = append(bgLines, "")
	}
	if len(bgLines) > bgHeight {
		bgLines = bgLines[:bgHeight]
	}

	fgHeight := len(fgLines)
	fgWidth := 0
	for _, l := range fgLines {
		if w := lipgloss.Width(l); w > fgWidth {
			fgWidth = w
		}
	}

	yOff := (bgHeight - fgHeight) / 2
	if yOff < 0 {
		yOff = 0
	}
	xOff := (bgWidth - fgWidth) / 2
	if xOff < 0 {
		xOff = 0
	}

	for i, fgLine := range fgLines {
		bgIdx := yOff + i
		if bgIdx >= len(bgLines) {
			break
		}
		bgLine := bgLines[bgIdx]
		bgLines[bgIdx] = spliceLineAnsi(bgLine, fgLine, xOff, bgWidth)
	}

	return strings.Join(bgLines, "\n")
}

func spliceLineAnsi(bgLine, fgLine string, xOff, totalWidth int) string {
	bgVis := lipgloss.Width(bgLine)
	if bgVis < totalWidth {
		bgLine += strings.Repeat(" ", totalWidth-bgVis)
	}

	fgWidth := lipgloss.Width(fgLine)

	var prefix, suffix strings.Builder
	visPos := 0
	runes := []rune(bgLine)

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\x1b' {
			escStart := i
			for i < len(runes) && runes[i] != 'm' {
				i++
			}
			escSeq := string(runes[escStart : i+1])
			if visPos < xOff {
				prefix.WriteString(escSeq)
			} else if visPos >= xOff+fgWidth {
				suffix.WriteString(escSeq)
			}
			continue
		}
		if visPos < xOff {
			prefix.WriteRune(ch)
		} else if visPos >= xOff+fgWidth {
			suffix.WriteRune(ch)
		}
		visPos++
	}

	return prefix.String() + fgLine + suffix.String()
}

func (m Model) padContent(lines []string, height int) string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) projectLines() []string {
	if len(m.store.Projects) == 0 {
		return []string{normalStyle.Render("No projects yet. Press a to create one.")}
	}

	lines := make([]string, 0, len(m.store.Projects))
	for i, p := range m.store.Projects {
		text := fmt.Sprintf("%s (%d)", p.Name, len(p.Todos))
		if i == m.projectCursor {
			lines = append(lines, selectedStyle.Render("▶ "+text))
		} else {
			lines = append(lines, normalStyle.Render("  "+text))
		}
	}
	return lines
}

func (m Model) todoLines() []string {
	p := m.currentProject()
	if p == nil {
		return []string{normalStyle.Render("Select or create a project.")}
	}
	if len(p.Todos) == 0 {
		return []string{normalStyle.Render("No todos yet. Press a to add one.")}
	}

	lines := make([]string, 0, len(p.Todos))
	for i, t := range p.Todos {
		prefix := "  "
		if i == m.todoCursor {
			prefix = "▶ "
		}
		box := "[ ]"
		if t.Completed {
			box = "[x]"
		}

		text := fmt.Sprintf("%s%s %s", prefix, box, t.Title)
		if t.Link != "" {
			text += " 🔗"
		}
		if t.Completed {
			lines = append(lines, completedStyle.Render(text))
		} else if i == m.todoCursor {
			lines = append(lines, selectedStyle.Render(text))
		} else {
			lines = append(lines, normalStyle.Render(text))
		}
	}
	return lines
}
