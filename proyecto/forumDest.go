package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type Hub struct {
	mu      sync.RWMutex
	connMu  sync.Mutex
	clients map[uint32]string
}


var buffers = sync.Pool{
	New: func() any {
		return make([]byte, 4096)
	},
}


func (h *Hub) post(msg string, channelID uint32, conn net.Conn, flags byte, isStealth bool) {
	h.mu.RLock()
	name, ok := h.clients[channelID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	txt := fmt.Sprintf("%s: %s", name, msg)
	payload := []byte(txt)
	h.mu.RUnlock()

	header := make([]byte, HeaderSize)
	header[VersionPos] = V1
	header[FlagsPos] = flags
	header[DestPos] = DestForum
	binary.BigEndian.PutUint32(header[LengthPos:LengthPos+4], uint32(len(payload)))

	h.mu.RLock()
	defer h.mu.RUnlock()
	for destID := range h.clients {
		if destID != channelID {
			binary.BigEndian.PutUint32(header[ChannelIdPos:ChannelIdPos+4], destID)
			h.connMu.Lock()
			packet := append(header, payload...)
			conn.Write(packet)
			h.connMu.Unlock()
		}
	}
}


func (h *Hub) logOff(channelId uint32) {
	h.mu.Lock()
	name, ok := h.clients[channelId]
	if ok {
		delete(h.clients, channelId)
		fmt.Printf("[DESCONEXIÓN] Usuario '%s' (ID %d) liberado.\n", name, channelId)
	}
	h.mu.Unlock()
}


func handleForum(conn net.Conn, hub *Hub) {
	header := make([]byte, HeaderSize)
	for {
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}

		flags := header[FlagsPos]
		channelId := binary.BigEndian.Uint32(header[ChannelIdPos : ChannelIdPos+4])
		
		if (flags & FIN) != 0 {
			hub.logOff(channelId)
			return 
		}

		plSize := binary.BigEndian.Uint32(header[LengthPos : LengthPos+4])
		isStealth := (flags & Stealth) != 0

		pBuffer := buffers.Get().([]byte)
		original := pBuffer
		if uint32(len(pBuffer)) < plSize {
			pBuffer = make([]byte, plSize)
		}

		if _, err := io.ReadFull(conn, pBuffer[:plSize]); err != nil {
			buffers.Put(original)
			return
		}

		msg := string(pBuffer[:plSize])

		hub.mu.RLock()
		_, registered := hub.clients[channelId]
		hub.mu.RUnlock()

		if !registered {
			hub.mu.Lock()
			hub.clients[channelId] = msg
			hub.mu.Unlock()
			fmt.Printf("[REGISTRO] ID %d asociado a '%s'\n", channelId, msg)
		} else {
			hub.post(msg, channelId, conn, flags, isStealth)
		}

		buffers.Put(original)
	}
}


func main() {
	hub := &Hub{clients: make(map[uint32]string)}
	listener, err := net.Listen("tcp", ":8083")
	if err != nil {
		log.Fatalf("Error listener foro: %v", err)
	}
	fmt.Println("Foro escuchando en :8083...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleForum(conn, hub)
	}
}
