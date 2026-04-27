package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Clients struct {
	clients map[uint32]net.Conn
	muCli   sync.RWMutex
}

type Destiny struct {
	conn   net.Conn
	muDest sync.Mutex
}

type Id struct {
	nextId uint32
	muId   sync.Mutex
}

var (
	clients  = &Clients{clients: make(map[uint32]net.Conn)}
	connDest = &Destiny{}
	pool     = sync.Pool{
		New: func() any {
			return make([]byte, 4096)
		},
	}
	nextId = &Id{nextId: 1}
)

func plog(prefix string, msg string, stealth bool) {
	if !stealth {
		log.Printf("[%s] %s", prefix, msg)
	}
}

func kill(myID uint32) {
	finHeader := make([]byte, HeaderSize)
	finHeader[VersionPos] = V1
	finHeader[FlagsPos] = FIN
	finHeader[DestPos] = DestForum
	binary.BigEndian.PutUint32(finHeader[ChannelIdPos:ChannelIdPos+4], myID)

	connDest.muDest.Lock()
	if connDest.conn != nil {
		connDest.conn.Write(finHeader)
	}
	connDest.muDest.Unlock()

	clients.muCli.Lock()
	if c, ok := clients.clients[myID]; ok {
		c.Close()
		delete(clients.clients, myID)
		plog("PROXY", fmt.Sprintf("Cliente %d desconectado y limpiado.", myID), false)
	}
	clients.muCli.Unlock()
}

func handleConn(conn net.Conn) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close()
		return
	}

	// Extraer flags iniciales para el handshake
	initialFlags := header[FlagsPos]
	isStealth := (initialFlags & Stealth) != 0

	plSize := binary.BigEndian.Uint32(header[LengthPos : LengthPos+4])
	payload := make([]byte, plSize)
	if plSize > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			conn.Close()
			return
		}
	}

	connDest.muDest.Lock()
	if connDest.conn == nil {
		connDest.muDest.Unlock()
		conn.Write([]byte{ErrServiceDown})
		conn.Close()
		return
	}
	connDest.muDest.Unlock()

	nextId.muId.Lock()
	myID := nextId.nextId
	nextId.nextId++
	nextId.muId.Unlock()

	clients.muCli.Lock()
	clients.clients[myID] = conn
	clients.muCli.Unlock()

	plog("PROXY", fmt.Sprintf("Handshake exitoso. ID %d asignado a '%s'", myID, string(payload)), isStealth)

	defer kill(myID)

	connDest.muDest.Lock()
	binary.BigEndian.PutUint32(header[ChannelIdPos:ChannelIdPos+4], myID)
	packet := append(header, payload...)
	connDest.conn.Write(packet)
	connDest.muDest.Unlock()
	
	conn.Write([]byte{ErrNone})
	
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	for {
		if _, err := io.ReadFull(conn, header); err != nil {
			break
		}

		flags := header[FlagsPos]
		isFin := (flags & FIN) != 0
		isKeepAlive := (flags & KeepAlive) != 0
		isStealth = (flags & Stealth) != 0
		plSize := binary.BigEndian.Uint32(header[LengthPos : LengthPos+4])

		if isKeepAlive {
			conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			plog("PROXY", fmt.Sprintf("KeepAlive recibido del ID %d. Timeout renovado.", myID), isStealth)
			
			if plSize == 0 && !isFin {
				continue
			}
		} else {
			conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		}

		pBuffer := pool.Get().([]byte)
		original := pBuffer
		if uint32(len(pBuffer)) < plSize {
			pBuffer = make([]byte, plSize)
		}

		if plSize > 0 {
			if _, err := io.ReadFull(conn, pBuffer[:plSize]); err != nil {
				pool.Put(original)
				break
			}
			plog("PROXY", fmt.Sprintf("Mensaje reenviado del ID %d (%d bytes)", myID, plSize), isStealth)
		}

		binary.BigEndian.PutUint32(header[ChannelIdPos:ChannelIdPos+4], myID)

		connDest.muDest.Lock()
		pckt:= append(header, pBuffer[:plSize]...)
		connDest.conn.Write(pckt)
		connDest.muDest.Unlock()

		pool.Put(original)

		if isFin {
			plog("PROXY", fmt.Sprintf("Flag FIN recibida del ID %d. Iniciando cierre.", myID), isStealth)
			break
		}
	}
}

func dealer() {
	headerBuff := make([]byte, HeaderSize)
	for {
		if _, err := io.ReadFull(connDest.conn, headerBuff); err != nil {
			plog("PROXY", fmt.Sprintf("Conexión con el destino perdida: %v", err), false)
			return
		}

		channelId := binary.BigEndian.Uint32(headerBuff[ChannelIdPos : ChannelIdPos+4])
		plSize := binary.BigEndian.Uint32(headerBuff[LengthPos : LengthPos+4])

		pBuffer := pool.Get().([]byte)
		original := pBuffer
		if uint32(len(pBuffer)) < plSize {
			pBuffer = make([]byte, plSize)
		}

		if plSize > 0 {
			if _, err := io.ReadFull(connDest.conn, pBuffer[:plSize]); err != nil {
				pool.Put(original)
				return
			}
		}

		clients.muCli.RLock()
		clientConn, ok := clients.clients[channelId]
		clients.muCli.RUnlock()

		if ok {
			packet := append(headerBuff, pBuffer[:plSize]...)
			clientConn.Write(packet)
		}	
		pool.Put(original)
	}
}


func supervisor() {
    for {

        dealer()

		reconnected := false
		plog("PROXY", "Foro desconectado. Limpiando clientes...", false)

        clients.muCli.RLock()
        var ids []uint32
        for id := range clients.clients {
            ids = append(ids, id)
        }
        clients.muCli.RUnlock()

        for _, id := range ids {
            kill(id)
        }

        for i := 1; i <= 30; i++ {
			plog("PROXY", fmt.Sprintf("Reintentando conexión al foro... intento %d", i), false)
            
            conn, err := net.DialTimeout("tcp", DestMap[DestForum], 5*time.Second)
            if err == nil {
        
                connDest.muDest.Lock()
                connDest.conn = conn
                connDest.muDest.Unlock()
                
				reconnected = true
                plog("PROXY", "Reconectado al foro. Relanzando dealer.", false)
                break
            }

            wait := time.Duration(i) * time.Second
            plog("PROXY", fmt.Sprintf("Foro no disponible. Esperando %v...", wait), false)
            time.Sleep(wait)
        }

		if !reconnected { log.Fatal("No se pudo reconectar al foro después de 30 intentos. Cerrando proxy.") }
    }
}


func main() {
	var err error
	connDest.conn, err = net.DialTimeout("tcp", DestMap[DestForum], 5*time.Second)
	if err != nil {
		log.Fatalf("No se pudo conectar al foro: %v", err)
	}
	defer connDest.conn.Close()

	go supervisor()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		plog("PROXY", "Apagando proxy. Avisando a clientes y servicios.", false)
		
		clients.muCli.RLock()
		var ids []uint32
		for id := range clients.clients {
			ids = append(ids, id)
		}
		clients.muCli.RUnlock()

		for _, id := range ids {
			kill(id)
		}
		os.Exit(0)
	}()

	listener, err := net.Listen("tcp", ProxyPort)
	if err != nil {
		log.Fatalf("Error al abrir listener: %v", err)
	}
	plog("PROXY", fmt.Sprintf("Escuchando en %s", ProxyPort), false)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn)
	}
}
