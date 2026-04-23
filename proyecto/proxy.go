package main

import (
	"crypto/des"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func plog(prefix string, msg string, stealth bool) {
	if !stealth {
		log.Printf("[%s] %s", prefix, msg)
	}
}

func handleConn(conn net.Conn) {
	clientAdrr := conn.RemoteAddr().String()

	// Setup de la conexion (Manejo conexion original y abrir el header)
	defer ErrorHandler(conn.Close())

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(conn, header)
	ErrorHandler(err)	

	conn.SetReadDeadline(time.Time{})

	// Flags byte 2
	isStealth := (header[FlagsPos] & Stealth) != 0
	hasKeepAlive := (header[FlagsPos] & KeepAlive) != 0
	destKey := header[DestPos]
	version := header[VersionPos]
	plSize := binary.BigEndian.Uint32(header[LengthPos:LengthPos+4])

	// Verificación de parametros del header

	// Versión byte 1
	switch header[VersionPos] {
	case V1:
		plog(clientAdrr, fmt.Sprintf("Versión %x aceptada", version), isStealth)
	default:
		plog(clientAdrr, fmt.Sprintf("Error: Versión %x no soportada", version), isStealth)
		_, err := conn.Write([]byte{ErrInvalidVersion})
		ErrorHandler(err)

		return
	}

	// Destino byte 3
	dest, ok := DestMap[destKey]
	if !ok {
		plog(clientAdrr, fmt.Sprintf("Error: Destino %x no encontrado", destKey), isStealth)
		return
	}

	plog(clientAdrr, fmt.Sprintf("Conectando a %s", dest), isStealth)

	fPl := make([]byte, plSize)
	ErrorHandler(io.ReadFull(conn, fPl))

	destConn, err := net.DialTimeout("tcp", dest, 5*time.Second)
	if err != nil {
		_, err := conn.Write([]byte{ErrServiceDown})
		if err != nil {
			return
		}

		return
	}

	defer ErrorHandler(destConn.Close())

	destConn.Write(header)
	destConn.Write(fPl)

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
	go forward(destConn, conn, &wg, isStealth, "BACKWARD")

	wg.Wait()
}

func forward(src, dest net.Conn, wg *sync.WaitGroup, stealth bool, prefix string) {
	defer wg.Done()

	headerBuff := make([]byte, HeaderSize)

	for {
		ErrorHandler(io.ReadFull(src, headerBuff))
	
		size := binary.BigEndian.Uint32(headerBuff[LengthPos:LengthPos+4])

		payloadBuff := make([]byte, size)
		ErrorHandler(io.ReadFull(src,  payloadBuff))

		dest.Write(headerBuff)
		dest.Write(payloadBuff)

		plog(prefix, fmt.Sprintf("Paquete reenvaiado (%d bytes)", size), stealth)

	}
}

func main() {
	listener, err := net.Listen("tcp", ProxyPort)
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
