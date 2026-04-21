package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"bufio"

	"proyecto"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	schVersion byte = 0x01
	schFlags   byte = 0x00
	schDest byte = 0x03
)

func connProxy() (net.Conn, error) {
	conn, err := net.Dial("tcp", proyecto.ProxyPort)
	if err != nil { return nil, fmt.Errorf("no se pudo conectar al proxy: %w", err)}

	header := []byte{schVersion,schFlags,schDest}
	_, err = conn.Write(header)
	if err != nil {return nil, err}

	resp := make([]byte,1)
	_, err = io.ReadFull(conn, resp)
	if err != nil || resp[0] != 0x00 {return nil, fmt.Errorf("proxy rechazo la conexion")}

	return conn, nil
}

func readMsg(conn net.Conn, p *tea.Program) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		p.Send(msgReceived{text: scanner.Text()})
	}
	p.Send(msgDisconnected{})
}

func main() {

	conn, err := connProxy()
	if err != nil {
		return
	}
	m := initialModel(conn)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go readMsg(conn, p)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Hubo un error en la interfaz: %v", err)
		os.Exit(1)
	}
}
