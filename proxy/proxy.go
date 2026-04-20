package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"proyecto"
)

func plog(prefix string, msg string, stealth bool) {
	if !stealth {
		// Usamos log.Printf para formatear el prefijo y el mensaje
		log.Printf("[%s] %s", prefix, msg)
	}
}

func handleConn(conn net.Conn) {
	clientAdrr := conn.RemoteAddr().String()

	// Setup de la conexion (Manejo conexion original y abrir el header)
	defer func() {
		err := conn.Close()
		if err != nil {
			return
		}
	}()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	header := make([]byte, proyecto.HeaderSize)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return
	}

	conn.SetReadDeadline(time.Time{})

	// Flags byte 2
	isStealth := (header[proyecto.FlagsPos] & proyecto.Stealth) != 0
	hasKeepAlive := (header[proyecto.FlagsPos] & proyecto.KeepAlive) != 0

	// Verificación de parametros del header

	// Versión byte 1
	switch header[proyecto.VersionPos] {
	case proyecto.V1:
		plog(clientAdrr, fmt.Sprintf("Versión %x aceptada", header[proyecto.VersionPos]), isStealth)
	default:
		plog(clientAdrr, fmt.Sprintf("Error: Versión %x no soportada", header[proyecto.VersionPos]), isStealth)
		_, err := conn.Write([]byte{proyecto.ErrInvalidVersion})
		if err != nil {
			return
		}
		return
	}

	// Destino byte 3
	dest, ok := proyecto.DestMap[header[proyecto.DestPos]]
	if !ok {
		plog(clientAdrr, fmt.Sprintf("Error: Destino %x no encontrado", header[proyecto.DestPos]), isStealth)
		return
	}

	plog(clientAdrr, fmt.Sprintf("Conectando a %s", dest), isStealth)

	destConn, err := net.DialTimeout("tcp", dest, 5*time.Second)
	if err != nil {
		_, err := conn.Write([]byte{proyecto.ErrServiceDown})
		if err != nil {
			return
		}

		return
	}
	_, err = conn.Write([]byte{0x00})
	if err != nil {
		return
	}

	defer func() {
		err := destConn.Close()
		if err != nil {
			return
		}
	}()

	if hasKeepAlive {

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
		}

		if tcpDest, ok := destConn.(*net.TCPConn); ok {
			tcpDest.SetKeepAlive(true)
		}

	}

	var wg sync.WaitGroup
	wg.Add(2)

	go forward(conn, destConn, &wg, isStealth, clientAdrr)
	go forward(destConn, conn, &wg, isStealth, clientAdrr)

	wg.Wait()
}

func forward(dest, source net.Conn, wg *sync.WaitGroup, isStealth bool, clientAddr string) {
	defer wg.Done()

	written, err := io.Copy(dest, source)
	if err != nil {
		plog(clientAddr, fmt.Sprintf("%d", written), isStealth)
	}

	if tcp, ok := dest.(*net.TCPConn); ok {
		tcp.CloseWrite()
	}
}

func main() {
	listener, err := net.Listen("tcp", proyecto.ProxyPort)
	if err != nil {
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go handleConn(conn)
	}
}
