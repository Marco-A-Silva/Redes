package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

type Hub struct {
	mu      sync.Mutex
	clients map[net.Conn]string
	users 	map[string]net.Conn
}

func (h *Hub) post(msg string, sender net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if conn != sender {
			ErrorHandler(fmt.Fprintln(conn, msg))
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
	ErrorHandler(conn.Close())
}

func handleForum(conn net.Conn, hub *Hub) {
	defer hub.logOff(conn)

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	nombre := scanner.Text()
	hub.clients[conn] = nombre
	hub.users[nombre] = conn
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
		users: make(map[string]net.Conn),
	}

	listener, err := net.Listen("tcp", DestMap[DestForum])
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
