package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"proyecto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	msgReceived     struct{ text string }
	msgDisconnected struct{}
)

var (
	styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Padding(0, 1)
	styleMsgArea   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	styleInputArea = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	styleName      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleSystem    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)

type model struct {
	conn          net.Conn
	loggedIn      bool
	name          string
	input         string
	messages      []string
	width, height int
	err           error
}

func initialModel() model {
	return model{}
}

// receiveMessageCmd implementa la lectura estricta del protocolo SCH
func receiveMessageCmd(conn net.Conn) tea.Cmd {
	return func() tea.Msg {
		// 1. Leer el Header (11 bytes)
		header := make([]byte, proyecto.HeaderSize)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			return msgDisconnected{}
		}

		// 2. Extraer el tamaño del mensaje (LengthPos es 7)
		length := binary.BigEndian.Uint32(header[proyecto.LengthPos : proyecto.LengthPos+4])

		// 3. Leer el Payload (el texto real)
		payload := make([]byte, length)
		if length > 0 {
			_, err = io.ReadFull(conn, payload)
			if err != nil {
				return msgDisconnected{}
			}
		}

		return msgReceived{text: string(payload)}
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case msgReceived:
		m.messages = append(m.messages, strings.TrimSpace(msg.text))
		// Volvemos a pedir un comando de lectura para mantener el bucle activo
		return m, receiveMessageCmd(m.conn)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit

		case tea.KeyEnter:
			if !m.loggedIn {
				m.name = strings.TrimSpace(m.input)
				if m.name == "" {
					return m, nil
				}

				conn, err := connectToProxy(proyecto.DestForum)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.conn = conn
				m.loggedIn = true
				m.input = ""

				// Enviamos el nombre de usuario como primer paquete
				sendPacket(m.conn, proyecto.DestForum, []byte(m.name+"\n"))

				// Iniciamos la cadena de lectura de red
				return m, receiveMessageCmd(m.conn)
			}

			if strings.TrimSpace(m.input) != "" {
				fullMsg := fmt.Sprintf("%s: %s\n", m.name, m.input)
				m.messages = append(m.messages, fmt.Sprintf("> %s", m.input))
				sendPacket(m.conn, proyecto.DestForum, []byte(fullMsg))
				m.input = ""
			}

		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyRunes, tea.KeySpace:
			m.input += msg.String()
		}

	case msgDisconnected:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Cargando..."
	}

	if !m.loggedIn {
		title := styleHeader.Render(" PROYECTO SCH ") + "\n\n"
		prompt := "Introduce tu nombre de usuario:\n"
		input := styleInputArea.Width(30).Render(m.input)
		content := lipgloss.JoinVertical(lipgloss.Center, title, prompt, input)
		if m.err != nil {
			content += "\n\n" + styleSystem.Foreground(lipgloss.Color("196")).Render(m.err.Error())
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	header := styleHeader.Render(fmt.Sprintf("FORO SCH — %s", styleName.Render(m.name)))
	msgHeight := m.height - 10
	if msgHeight < 2 {
		msgHeight = 2
	}

	visibleMsgs := m.messages
	if len(visibleMsgs) > msgHeight {
		visibleMsgs = m.messages[len(m.messages)-msgHeight:]
	}

	msgs := styleMsgArea.Width(m.width - 4).Height(msgHeight).Render(strings.Join(visibleMsgs, "\n"))
	input := styleInputArea.Width(m.width - 4).Render("Mensaje: " + m.input)

	return lipgloss.JoinVertical(lipgloss.Left, header, msgs, input)
}
