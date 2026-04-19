package main

// ACH es AcademyConnectionHandler

const ProxyPort = ":8080"

// Verisionado
const (
	V1 byte = 0x01
)

// Flags
const (
	KeepAlive byte = 0x01
	Prio      byte = 0x02
	Stealth   byte = 0x04
	Checksum  byte = 0x08
)

// Destinos
const (
	DestMsg   = 0x01
	DestVid   = 0x02
	DestForum = 0x03
)

var DestMap = map[byte]string{
	DestMsg:   "127.0.0.1:8081",
	DestVid:   "127.0.0.1:8082",
	DestForum: "127.0.0.1:8083",
}

// Estructura
const (
	VersionPos = 0
	FlagsPos   = 1
	DestPos    = 2
	HeaderSize = 3
)

// Debugging
const (
	ErrNone           = 0x00
	ErrInvalidVersion = 0xF0
	ErrDestNotFound   = 0xF1
	ErrServiceDown    = 0xF2
	ErrInternal       = 0xFF
)
