package main

import (
	"fmt"
	"net"
	"strings"

	"proyecto" // Asegurate que el go.mod en la raíz diga 'module proyecto'

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Estilos Globales
// ---------------------------------------------------------------------------
var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			Padding(0, 1)

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

type (
	msgReceived     struct{ text string }
	msgDisconnected struct{}
)

type screen int

const (
	screenMenu screen = iota
	screenLogin
	screenChat
)

type model struct {
	screen       screen
	conn         net.Conn
	destSelected byte
	name         string
	input        string
	messages     []string
	width        int
	height       int
	disconnected bool
}

func initialModel() model {
	return model{
		screen: screenMenu,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case msgReceived:
		m.messages = append(m.messages, msg.text)
		return m, startReader(m.conn)

	case msgDisconnected:
		m.disconnected = true
		m.messages = append(m.messages, styleDisconnected.Render("— conexión cerrada —"))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.screen == screenLogin {
				return m.handleLoginTransition()
			}
			if m.screen == screenChat {
				return m.handleChatEnter()
			}

		case tea.KeyRunes:
			if m.screen == screenMenu {
				switch msg.String() {
				case "1":
					m.destSelected = proyecto.DestMsg
				case "2":
					m.destSelected = proyecto.DestVid
				case "3":
					m.destSelected = proyecto.DestForum
				}
				m.screen = screenLogin
				return m, nil
			}
			m.input += msg.String()

		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		}
	}
	return m, nil
}

func (m model) handleLoginTransition() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input)
	if name == "" {
		return m, nil
	}
	m.name = name
	m.input = ""

	c, err := connectToProxy(m.destSelected)
	if err != nil {
		m.messages = append(m.messages, styleDisconnected.Render("Error: "+err.Error()))
		return m, nil
	}

	m.conn = c
	m.screen = screenChat
	fmt.Fprintf(m.conn, "%s se unió al chat\n", name)

	return m, startReader(m.conn)
}

func (m model) handleChatEnter() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input)
	if text == "" || m.disconnected {
		return m, nil
	}
	// Enviar al proxy
	fmt.Fprintf(m.conn, "%s: %s\n", m.name, text)
	m.input = ""
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Cargando..."
	}
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenLogin:
		return m.viewLogin()
	default:
		return m.viewChat()
	}
}

func (m model) viewMenu() string {
	title := styleHeader.Render("ACADEMY SCH CLIENT")
	options := "\nSeleccioná un servicio:\n\n" +
		"  [1] Chat General\n" +
		"  [2] Streaming\n" +
		"  [3] Foros\n\n" +
		styleSystem.Render("Presioná un número para continuar")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, title+options)
}

func (m model) viewLogin() string {
	title := styleHeader.Render("CONFIGURACIÓN DE PERFIL")
	prompt := fmt.Sprintf("\nConectando a Destino: %x\nIngresá tu nombre:\n\n", m.destSelected)
	input := styleInputArea.Width(30).Render(styleInputLabel.Render("> ") + m.input + "█")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, title+prompt+input)
}

func (m model) viewChat() string {
	// 1. Cálculos de espacio
	headerHeight := 2
	inputHeight := 3
	footerHeight := 1
	// Espacio para los mensajes (restando bordes y cabeceras)
	msgHeight := m.height - headerHeight - inputHeight - footerHeight - 2
	if msgHeight < 1 {
		msgHeight = 1
	}

	innerWidth := m.width - 4

	// 2. Header
	header := styleHeader.Render(fmt.Sprintf("SCH Chat — %s", styleName.Render(m.name)))

	// 3. Historial de Mensajes
	visibleMsgs := m.messages
	if len(visibleMsgs) > msgHeight {
		visibleMsgs = visibleMsgs[len(visibleMsgs)-msgHeight:]
	}
	msgContent := strings.Join(visibleMsgs, "\n")

	// Relleno para mantener la altura de la caja constante
	lineCount := strings.Count(msgContent, "\n") + 1
	if msgContent == "" {
		lineCount = 0
	}
	for i := lineCount; i < msgHeight; i++ {
		msgContent += "\n"
	}

	msgBox := styleMsgArea.Width(innerWidth).Height(msgHeight).Render(msgContent)

	// 4. Input Box
	inputContent := styleInputLabel.Render(m.name+" > ") + m.input + "█"
	inputBox := styleInputArea.Width(innerWidth).Render(inputContent)

	// 5. Footer / Ayuda
	hint := styleSystem.Render("  [Esc] Salir | [Enter] Enviar mensaje")
	if m.disconnected {
		hint = styleDisconnected.Render("  CONEXIÓN PERDIDA")
	}

	// 6. Ensamblado
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		msgBox,
		inputBox,
		hint,
	)
}
