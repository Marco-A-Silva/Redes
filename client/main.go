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

func sendPacket(conn net.Conn, dest byte, flags byte, payload []byte) error {
	header := make([]byte, proyecto.HeaderSize)
	header[proyecto.VersionPos] = proyecto.V1
	header[proyecto.FlagsPos] = flags
	header[proyecto.DestPos] = dest
	binary.BigEndian.PutUint32(header[proyecto.LengthPos:proyecto.LengthPos+4], uint32(len(payload)))

	if _, err := conn.Write(header); err != nil { return err }
	if len(payload) > 0 { _, err := conn.Write(payload); return err }
	return nil
}

func connectToProxy(dest byte, initialFlags byte, name string) (net.Conn, error) {
	conn, err := net.Dial("tcp", proyecto.ProxyPort)
	if err != nil { return nil, err }

	// Ahora enviamos el nombre como payload inicial [cite: 3]
	payload := []byte(name)
	if err := sendPacket(conn, dest, initialFlags, payload); err != nil {
		conn.Close()
		return nil, err
	}

	fmt.Println("Intentando handshake...")
	resp := make([]byte, 1)
	// El Proxy responderá 0x00 después de recibir este primer paquete [cite: 4]
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake fallido o rechazado: %v", err)
	}
	
	fmt.Println("Conexión aceptada por el Proxy")
	return conn, nil
}

func main() {
	m := initialModel()
	gProgram = tea.NewProgram(m, tea.WithAltScreen())
	if _, err := gProgram.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
