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
	styleCmdLine   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("51")).Foreground(lipgloss.Color("51")).Padding(0, 1)
	styleFlagOn    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	styleFlagOff   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	gProgram *tea.Program
)

type model struct {
	conn          net.Conn
	loggedIn      bool
	name          string
	input         string
	cmdInput      string
	showCmdline   bool
	messages      []string
	width, height int
	flags         byte
}

func initialModel() model {
	return model{flags: 0x00}
}

func (m *model) processCommand() {
	parts := strings.Fields(strings.ToLower(m.cmdInput))
	for _, cmd := range parts {
		switch cmd {
		case "fin":
			m.flags ^= proyecto.FIN
		case "stealth":
			m.flags ^= proyecto.Stealth
		case "alive":
			m.flags ^= proyecto.KeepAlive
		}
	}
}

func listenMessages(conn net.Conn) {
	for {
		header := make([]byte, proyecto.HeaderSize)
		if _, err := io.ReadFull(conn, header); err != nil {
			gProgram.Send(msgDisconnected{})
			return
		}
		length := binary.BigEndian.Uint32(header[proyecto.LengthPos : proyecto.LengthPos+4])
		payload := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				gProgram.Send(msgDisconnected{})
				return
			}
		}
		gProgram.Send(msgReceived{text: string(payload)})
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case msgReceived:
		m.messages = append(m.messages, strings.TrimSpace(msg.text))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.conn != nil { m.conn.Close() }
			return m, tea.Quit
		case tea.KeyEsc:
			if m.showCmdline {
				m.showCmdline = false
				m.cmdInput = ""
				return m, nil
			}
			if m.conn != nil { m.conn.Close() }
			return m, tea.Quit
		}

		if m.showCmdline {
			switch msg.Type {
			case tea.KeyEnter:
				m.processCommand()
				m.showCmdline = false
				m.cmdInput = ""
			case tea.KeyBackspace:
				if len(m.cmdInput) > 0 { m.cmdInput = m.cmdInput[:len(m.cmdInput)-1] }
			case tea.KeyRunes, tea.KeySpace:
				m.cmdInput += msg.String()
			}
			return m, nil
		}

		if msg.String() == "/" {
			m.showCmdline = true
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(m.input)
			if !m.loggedIn {
    			if input == "" { return m, nil }
    				m.name = input
   		 			// Pasamos m.name al conectar [cite: 3]
    				conn, err := connectToProxy(proyecto.DestForum, m.flags, m.name) 
    				if err != nil {
        				m.messages = append(m.messages, "Error: Proxy fuera de línea")
        				return m, nil
    				}
    				m.conn = conn
    				m.loggedIn = true
    				m.input = ""
    				go listenMessages(m.conn)
    				return m, nil
			}			
			if input != "" {
				sendPacket(m.conn, proyecto.DestForum, m.flags, []byte(input))
				m.messages = append(m.messages, "> "+input)
				m.input = ""
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 { m.input = m.input[:len(m.input)-1] }
		case tea.KeyRunes, tea.KeySpace:
			m.input += msg.String()
		}

	case msgDisconnected:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 { return "Cargando..." }

	renderFlag := func(name string, bit byte) string {
		if m.flags&bit != 0 { return styleFlagOn.Render("[ON] " + name) }
		return styleFlagOff.Render("[OFF] " + name)
	}

	fBar := fmt.Sprintf("%s | %s | %s",
		renderFlag("FIN", proyecto.FIN),
		renderFlag("STEALTH", proyecto.Stealth),
		renderFlag("ALIVE", proyecto.KeepAlive),
	)

	var toggleArea string
	if m.showCmdline {
		toggleArea = "\n" + styleCmdLine.Width(40).Render("toggle ❯ " + m.cmdInput)
	}

	txt := " PROYECTO SCH "
	if m.loggedIn { txt = " FORO SCH — " + m.name }
	header := styleHeader.Render(txt)

	if !m.loggedIn {
		content := lipgloss.JoinVertical(lipgloss.Center,
			header,
			"\nIntroduce tu nombre:",
			styleInputArea.Width(30).Render(m.input),
			"\n"+fBar,
			toggleArea)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	msgs := styleMsgArea.Width(m.width - 4).Height(m.height - 12).Render(strings.Join(m.messages, "\n"))
	input := styleInputArea.Width(m.width - 4).Render("Mensaje: " + m.input)

	return lipgloss.JoinVertical(lipgloss.Left, header, msgs, input, fBar, toggleArea)
}
