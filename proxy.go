package main

import (
	"io"
	"log"
	"net"
)

func handleConn(conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			return
		}
	}()

	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return
	}

	switch header[VersionPos] {
	case V1:
		log.Printf("Versión %x aceptada", header[VersionPos])
	default:
		log.Printf("Error: Versión %x no soportada", header[VersionPos])
		_, err := conn.Write([]byte{ErrInvalidVersion})
		if err != nil {
			return
		}
		return
	}

	dest, ok := DestMap[header[DestPos]]
	if !ok {
		log.Printf("Error: Destino %x no encontrado", header[DestPos])
		return
	}

	log.Printf("Conectando a %s", dest)

	destConn, err := net.Dial("tcp", dest)
	if err != nil {
		_, err := conn.Write([]byte{ErrServiceDown})
		if err != nil {
			return
		}

		return
	}
	defer func() {
		err := destConn.Close()
		if err != nil {
			return
		}
	}()

	go func() {
		_, err = io.Copy(conn, destConn)
		if err != nil {
			return
		}
	}()
	_, err = io.Copy(destConn, conn)
	if err != nil {
		return
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
