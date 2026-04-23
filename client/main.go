package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"proyecto"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	schVersion byte = 0x01
	schFlags   byte = 0x00
)

// sendPacket empaqueta y envía datos siguiendo el protocolo SCH de 11 bytes
func sendPacket(conn net.Conn, dest byte, payload []byte) error {
	header := make([]byte, 11)
	header[0] = schVersion
	header[1] = schFlags
	header[2] = dest
	
	binary.BigEndian.PutUint32(header[3:7], 0)
	binary.BigEndian.PutUint32(header[7:11], uint32(len(payload)))

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

func connectToProxy(dest byte) (net.Conn, error) {
	conn, err := net.Dial("tcp", "localhost"+proyecto.ProxyPort)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar al proxy: %w", err)
	}

	// Handshake inicial
	if err := sendPacket(conn, dest, []byte{}); err != nil {
		conn.Close()
		return nil, err
	}

	resp := make([]byte, 1)
	if _, err := io.ReadFull(conn, resp); err != nil || resp[0] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("proxy rechazó la conexión")
	}

	return conn, nil
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
