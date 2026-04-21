package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"proyecto"
)

func errorHandler(args ...any) {
	for _, args := range args {
		switch v:= args.(type) {
		case error:
			if v != nil {
				log.Printf("Error: %v", v)
			}
		case bool:
			if !v {
				log.Print("NOT OK")
			}
		default:
		}
	}
}

type Hub struct {
	mu      sync.Mutex
	clients map[net.Conn]string
}

func (h *Hub) post(msg string, sender net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if conn != sender {
			errorHandler(fmt.Fprintln(conn, msg))
		}
	}
}

func (h *Hub) register(conn net.Conn) {
	h.mu.Lock()
	h.clients[conn] = ""
	h.mu.Unlock()
}

func (h *Hub) logOff(conn net.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	errorHandler(conn.Close())
}

func handleForum(conn net.Conn, hub *Hub) {
	defer hub.logOff(conn)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		hub.post(text, conn)
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

func main() {
	hub := &Hub{
		clients: make(map[net.Conn]string),
	}

	listener, err := net.Listen("tcp", proyecto.DestMap[proyecto.DestForum])
	if err != nil {
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		hub.register(conn)
		go handleForum(conn, hub)
	}
}
