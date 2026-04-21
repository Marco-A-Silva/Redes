package main

import (
	"fmt"
	"net"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Mensajes internos de bubbletea
// ---------------------------------------------------------------------------

type (
	msgReceived     struct{ text string }
	msgDisconnected struct{}
)

// ---------------------------------------------------------------------------
// Estilos Refinados
// ---------------------------------------------------------------------------

var (
	// Estilo para el contenedor del diálogo de login
	styleLoginBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 3).
			Align(lipgloss.Center)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1)

	styleMsgArea = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	styleInputArea = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	styleInputLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)

	styleSystem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	styleName = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	styleDisconnected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)

// ---------------------------------------------------------------------------
// Estado y Modelo
// ---------------------------------------------------------------------------

type screen int

const (
	screenLogin screen = iota
	screenChat
)

type model struct {
	screen       screen
	conn         net.Conn
	name         string
	input        string
	messages     []string
	width        int
	height       int
	disconnected bool
}

func initialModel(conn net.Conn) model {
	return model{
		screen: screenLogin,
		conn:   conn,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case msgReceived:
		m.messages = append(m.messages, msg.text)

	case msgDisconnected:
		m.disconnected = true
		m.messages = append(m.messages, styleSystem.Render("— conexión cerrada por el servidor —"))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.screen == screenLogin {
				return m.handleLoginEnter()
			}
			return m.handleChatEnter()
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeySpace:
		case tea.KeyRunes:
			m.input += msg.String()
			
		}	
	}
	return m, nil
}

func (m model) handleLoginEnter() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input)
	if name == "" {
		return m, nil
	}
	m.name = name
	m.input = ""
	m.screen = screenChat
	m.messages = append(m.messages, styleSystem.Render(fmt.Sprintf("— conectado como %s —", name)))
	if m.conn != nil {
		fmt.Fprintf(m.conn, "%s se unió al chat\n", name)
	}
	return m, nil
}

func (m model) handleChatEnter() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input)
	if text == "" || m.disconnected {
		return m, nil
	}
	line := fmt.Sprintf("%s %s", ">", text)
	m.messages = append(m.messages, line)

	line = fmt.Sprintf("%s: %s", m.name, text)
	if m.conn != nil {
		fmt.Fprintf(m.conn, "%s\n", line)
	}
	m.input = ""
	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m model) View() string {
	if m.width == 0 {
		return "Cargando..."
	}
	if m.screen == screenLogin {
		return m.viewLogin()
	}
	return m.viewChat()
}

func (m model) viewLogin() string {
	// Definimos un ancho razonable para el diálogo, no toda la pantalla
	loginWidth := 40
	if m.width < loginWidth {
		loginWidth = m.width - 4
	}

	title := styleHeader.Render("SCH Forums")
	prompt := "Ingresá tu nombre:"

	// El input ahora tiene un ancho fijo dentro del diálogo centrado
	inputField := styleInputArea.Width(loginWidth - 10).Render(
		styleInputLabel.Render("> ") + m.input + "█",
	)

	hint := styleSystem.Render("Enter para conectar • Esc para salir")

	// Unimos todo verticalmente para que se mueva como un solo bloque
	uiBlock := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		prompt,
		"", // Espacio en blanco
		inputField,
		"", // Espacio en blanco
		hint,
	)

	// Colocamos el bloque en el centro exacto de la terminal
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		styleLoginBox.Render(uiBlock),
	)
}

func (m model) viewChat() string {
	inputHeight := 3
	headerHeight := 1
	msgHeight := m.height - inputHeight - headerHeight - 4
	if msgHeight < 1 {
		msgHeight = 1
	}

	innerWidth := m.width - 4

	header := styleHeader.MarginBottom(0).Render(fmt.Sprintf("SCH Forum  —  %s", styleName.Render(m.name)))

	visibleMsgs := m.messages
	if len(visibleMsgs) > msgHeight {
		visibleMsgs = visibleMsgs[len(visibleMsgs)-msgHeight:]
	}
	msgContent := strings.Join(visibleMsgs, "\n")

	lineCount := strings.Count(msgContent, "\n") + 1
	if msgContent == "" {
		lineCount = 0
	}
	for i := lineCount; i < msgHeight; i++ {
		msgContent += "\n"
	}
	msgBox := styleMsgArea.Width(innerWidth).Height(msgHeight).Render(msgContent)

	inputContent := styleInputLabel.Render("> ") + m.input + "█"
	inputBox := styleInputArea.Width(innerWidth).Render(inputContent)

	hint := styleSystem.Render("  Esc para salir")
	if m.disconnected {
		hint = styleDisconnected.Render("  desconectado")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		msgBox,
		inputBox,
		hint,
	)
}
