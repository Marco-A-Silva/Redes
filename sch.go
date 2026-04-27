package proyecto // wat

import (
	"log"
)

// ACH es AcademyConnectionHandler

const ProxyPort = ":8080"

// Verisionado
const (
	V1 byte = 0x01
)

// Flags
const (
	FIN byte = 0x01
	Prio byte = 0x02
	Stealth byte = 0x04
	Encrypt byte = 0x08
	KeepAlive byte = 0x10
)

// Destinos
const (
	DestVid   byte = 0x02
	DestForum byte = 0x03
)

var DestMap = map[byte]string{
	DestForum: "forum-service:8083",
}

// Estructura
const (
	VersionPos = 0
	FlagsPos   = 1
	DestPos    = 2
	ChannelIdPos = 3
	LengthPos = 7
	HeaderSize = 11
)

// Debugging
const (
	ErrNone           = 0x00
	ErrInvalidVersion = 0xF0
	ErrDestNotFound   = 0xF1
	ErrServiceDown    = 0xF2
	ErrInternal       = 0xFF
)

func ErrorHandler(args ...any) {
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
