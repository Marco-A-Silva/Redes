package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"proyecto" // Este nombre debe coincidir con el 'module' en go.mod

	tea "github.com/charmbracelet/bubbletea"
)

// connectToProxy ahora usa las constantes con el prefijo proyecto.
func connectToProxy(dest byte) (net.Conn, error) {
	// Usamos proyecto.ProxyPort (que es ":8080")
	conn, err := net.Dial("tcp", "127.0.0.1"+proyecto.ProxyPort)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar al proxy: %w", err)
	}

	// Mandar header SCH usando constantes del subpaquete
	header := []byte{proyecto.V1, 0x00, dest}
	_, err = conn.Write(header)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error mandando header: %w", err)
	}

	// Leer respuesta del proxy
	resp := make([]byte, 1)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error leyendo respuesta del proxy: %w", err)
	}

	// ErrNone también viene de proyecto
	if resp[0] != proyecto.ErrNone {
		conn.Close()
		return nil, fmt.Errorf("proxy rechazó la conexión: código %x", resp[0])
	}

	return conn, nil
}

// startReader maneja la entrada de datos del socket
func startReader(conn net.Conn) tea.Cmd {
	return func() tea.Msg {
		scanner := bufio.NewScanner(conn)
		if scanner.Scan() {
			return msgReceived{text: scanner.Text()}
		}
		return msgDisconnected{}
	}
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error en UI: %v\n", err)
		os.Exit(1)
	}
}
