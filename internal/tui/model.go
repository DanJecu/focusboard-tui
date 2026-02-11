package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/danjecu/focusboard-tui/internal/model"
	"github.com/danjecu/focusboard-tui/internal/storage"
)

type focusArea int

type inputMode int

type inputTarget int

const (
	focusProjects focusArea = iota
	focusTodos
)

const (
	modeNormal inputMode = iota
	modeInput
	modeConfirmDelete
)

const (
	targetNone inputTarget = iota
	targetAddProject
	targetEditProject
	targetAddTodo
	targetEditTodo
	targetSetLink
)

type Model struct {
	store         model.Store
	focus         focusArea
	mode          inputMode
	target        inputTarget
	projectCursor int
	todoCursor    int
	width         int
	height        int
	status        string
	statusErr     bool
	input         textarea.Model
	dataPath      string
	deleteMessage string
}

func New(path string) Model {
	s, err := storage.Load(path)
	status := "Ready"
	if err != nil {
		status = fmt.Sprintf("failed loading %s: %v", path, err)
	}

	ti := textarea.New()
	ti.Prompt = ""
	ti.CharLimit = 120
	ti.SetWidth(40)
	ti.SetHeight(3)
	ti.ShowLineNumbers = false

	m := Model{
		store:     s,
		focus:     focusProjects,
		mode:      modeNormal,
		target:    targetNone,
		status:    status,
		statusErr: err != nil,
		input:     ti,
		dataPath:  path,
	}
	m.clampCursors()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeInput {
			return m.handleInputKeys(msg)
		}
		if m.mode == modeConfirmDelete {
			return m.handleConfirmDeleteKeys(msg)
		}
		return m.handleNormalKeys(msg)
	default:
		return m, nil
	}
}

func (m Model) handleInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = modeNormal
		m.target = targetNone
		m.input.Blur()
		m.status = "Input cancelled"
		m.statusErr = false
		return m, nil
	case "ctrl+j", "shift+enter":
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(enterMsg)
		return m, cmd
	case "enter":
		m.commitInput()
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "ctrl+h", "left":
		m.focus = focusProjects
		m.status = "Focus: projects"
		m.statusErr = false
	case "ctrl+l", "right":
		if len(m.store.Projects) == 0 {
			m.status = "No projects available"
			m.statusErr = true
			return m, nil
		}
		m.focus = focusTodos
		m.status = "Focus: todos"
		m.statusErr = false
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "enter":
		m.handleEnter()
	case "a":
		m.beginAdd()
		if m.mode == modeInput {
			return m, textarea.Blink
		}
	case "e":
		m.beginEdit()
		if m.mode == modeInput {
			return m, textarea.Blink
		}
	case "d":
		m.deleteCurrent()
	case "l":
		if m.focus == focusTodos {
			p := m.currentProject()
			if p != nil && len(p.Todos) > 0 {
				m.mode = modeInput
				m.target = targetSetLink
				m.input.SetValue(p.Todos[m.todoCursor].Link)
				m.input.Placeholder = "https://github.com/..."
				m.input.Focus()
				m.status = "Set link"
				m.statusErr = false
				return m, textarea.Blink
			}
			m.status = "No todo to set link on"
			m.statusErr = true
		}
	case "o":
		if m.focus == focusTodos {
			p := m.currentProject()
			if p != nil && len(p.Todos) > 0 {
				link := p.Todos[m.todoCursor].Link
				if link != "" {
					return m, openURL(link)
				}
				m.status = "No link set (use l to add one)"
				m.statusErr = true
				return m, nil
			}
		}
	}

	m.clampCursors()
	return m, nil
}

func (m *Model) beginAdd() {
	if m.focus == focusProjects {
		m.mode = modeInput
		m.target = targetAddProject
		m.input.SetValue("")
		m.input.Placeholder = "Project name"
		m.input.Focus()
		m.status = "Add project"
		m.statusErr = false
		return
	}

	if m.currentProject() == nil {
		m.status = "Create a project first"
		m.statusErr = true
		return
	}
	m.mode = modeInput
	m.target = targetAddTodo
	m.input.SetValue("")
	m.input.Placeholder = "Todo title"
	m.input.Focus()
	m.status = "Add todo"
	m.statusErr = false
}

func (m *Model) beginEdit() {
	if m.focus == focusProjects {
		if len(m.store.Projects) == 0 {
			m.status = "No project to edit"
			m.statusErr = true
			return
		}
		m.mode = modeInput
		m.target = targetEditProject
		m.input.SetValue(m.store.Projects[m.projectCursor].Name)
		m.input.Placeholder = "Project name"
		m.input.Focus()
		m.status = "Edit project"
		m.statusErr = false
		return
	}

	p := m.currentProject()
	if p == nil || len(p.Todos) == 0 {
		m.status = "No todo to edit"
		m.statusErr = true
		return
	}
	m.mode = modeInput
	m.target = targetEditTodo
	m.input.SetValue(p.Todos[m.todoCursor].Title)
	m.input.Placeholder = "Todo title"
	m.input.Focus()
	m.status = "Edit todo"
	m.statusErr = false
}

func (m *Model) commitInput() {
	value := strings.TrimSpace(strings.ReplaceAll(m.input.Value(), "\n", " "))

	if m.target == targetSetLink {
		p := m.currentProject()
		if p != nil && len(p.Todos) > 0 {
			p.Todos[m.todoCursor].Link = value
			if value == "" {
				m.status = "Link cleared"
			} else {
				m.status = "Link saved"
			}
			m.statusErr = false
		}
		m.mode = modeNormal
		m.target = targetNone
		m.input.Blur()
		m.clampCursors()
		m.persist()
		return
	}

	if value == "" {
		m.mode = modeNormal
		m.target = targetNone
		m.input.Blur()
		m.status = "Empty input ignored"
		m.statusErr = true
		return
	}

	switch m.target {
	case targetAddProject:
		m.store.Projects = append(m.store.Projects, model.Project{Name: value, Todos: []model.Todo{}})
		m.projectCursor = len(m.store.Projects) - 1
		m.todoCursor = 0
		m.status = "Project created"
		m.statusErr = false
	case targetEditProject:
		if len(m.store.Projects) > 0 {
			m.store.Projects[m.projectCursor].Name = value
			m.status = "Project updated"
			m.statusErr = false
		}
	case targetAddTodo:
		p := m.currentProject()
		if p != nil {
			p.Todos = append(p.Todos, model.Todo{Title: value})
			m.todoCursor = len(p.Todos) - 1
			m.status = "Todo created"
			m.statusErr = false
		}
	case targetEditTodo:
		p := m.currentProject()
		if p != nil && len(p.Todos) > 0 {
			p.Todos[m.todoCursor].Title = value
			m.status = "Todo updated"
			m.statusErr = false
		}
	}

	m.mode = modeNormal
	m.target = targetNone
	m.input.Blur()
	m.clampCursors()
	m.persist()
}

func (m *Model) handleEnter() {
	if m.focus == focusProjects {
		if len(m.store.Projects) == 0 {
			m.status = "No projects available"
			m.statusErr = true
			return
		}
		m.focus = focusTodos
		m.clampCursors()
		m.status = fmt.Sprintf("Opened %q", m.store.Projects[m.projectCursor].Name)
		m.statusErr = false
		return
	}

	p := m.currentProject()
	if p == nil || len(p.Todos) == 0 {
		m.status = "No todo selected"
		m.statusErr = true
		return
	}
	p.Todos[m.todoCursor].Completed = !p.Todos[m.todoCursor].Completed
	if p.Todos[m.todoCursor].Completed {
		m.status = "Todo completed"
	} else {
		m.status = "Todo reopened"
	}
	m.statusErr = false
	m.persist()
}

func (m *Model) deleteCurrent() {
	if m.focus == focusProjects {
		if len(m.store.Projects) == 0 {
			m.status = "No project to delete"
			m.statusErr = true
			return
		}
		name := m.store.Projects[m.projectCursor].Name
		m.mode = modeConfirmDelete
		m.deleteMessage = fmt.Sprintf("Delete project %q? (y/n)", name)
		m.status = m.deleteMessage
		m.statusErr = false
		return
	}

	p := m.currentProject()
	if p == nil || len(p.Todos) == 0 {
		m.status = "No todo to delete"
		m.statusErr = true
		return
	}
	title := p.Todos[m.todoCursor].Title
	m.mode = modeConfirmDelete
	m.deleteMessage = fmt.Sprintf("Delete todo %q? (y/n)", title)
	m.status = m.deleteMessage
	m.statusErr = false
}

func (m *Model) confirmDelete() {
	if m.focus == focusProjects {
		if len(m.store.Projects) == 0 {
			return
		}
		name := m.store.Projects[m.projectCursor].Name
		m.store.Projects = append(m.store.Projects[:m.projectCursor], m.store.Projects[m.projectCursor+1:]...)
		m.clampCursors()
		m.focus = focusProjects
		m.status = fmt.Sprintf("Deleted project %q", name)
		m.statusErr = false
		m.persist()
		return
	}

	p := m.currentProject()
	if p == nil || len(p.Todos) == 0 {
		return
	}
	title := p.Todos[m.todoCursor].Title
	p.Todos = append(p.Todos[:m.todoCursor], p.Todos[m.todoCursor+1:]...)
	m.clampCursors()
	m.status = fmt.Sprintf("Deleted todo %q", title)
	m.statusErr = false
	m.persist()
}

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y", "enter":
		m.confirmDelete()
		m.mode = modeNormal
		m.deleteMessage = ""
		return m, nil
	case "n", "esc":
		m.mode = modeNormal
		m.deleteMessage = ""
		m.status = "Delete cancelled"
		m.statusErr = false
		return m, nil
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if m.focus == focusProjects {
		if len(m.store.Projects) == 0 {
			return
		}
		m.projectCursor += delta
		m.clampCursors()
		return
	}

	p := m.currentProject()
	if p == nil || len(p.Todos) == 0 {
		return
	}
	m.todoCursor += delta
	m.clampCursors()
}

func (m *Model) currentProject() *model.Project {
	if len(m.store.Projects) == 0 {
		return nil
	}
	if m.projectCursor < 0 || m.projectCursor >= len(m.store.Projects) {
		return nil
	}
	return &m.store.Projects[m.projectCursor]
}

func (m *Model) clampCursors() {
	if len(m.store.Projects) == 0 {
		m.projectCursor = 0
		m.todoCursor = 0
		return
	}

	if m.projectCursor < 0 {
		m.projectCursor = 0
	}
	if m.projectCursor >= len(m.store.Projects) {
		m.projectCursor = len(m.store.Projects) - 1
	}

	p := &m.store.Projects[m.projectCursor]
	if len(p.Todos) == 0 {
		m.todoCursor = 0
		return
	}

	if m.todoCursor < 0 {
		m.todoCursor = 0
	}
	if m.todoCursor >= len(p.Todos) {
		m.todoCursor = len(p.Todos) - 1
	}
}

func (m Model) inputTitle() string {
	switch m.target {
	case targetAddProject:
		return "New project"
	case targetEditProject:
		return "Edit project"
	case targetAddTodo:
		return "New todo"
	case targetEditTodo:
		return "Edit todo"
	case targetSetLink:
		return "Set link"
	default:
		return "Input"
	}
}

func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd string
		switch runtime.GOOS {
		case "darwin":
			cmd = "open"
		case "windows":
			cmd = "start"
		default:
			cmd = "xdg-open"
		}
		exec.Command(cmd, url).Start()
		return nil
	}
}

func (m *Model) persist() {
	if err := storage.Save(m.dataPath, m.store); err != nil {
		m.status = fmt.Sprintf("save failed: %v", err)
		m.statusErr = true
	}
}
