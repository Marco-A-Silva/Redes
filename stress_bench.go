package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	TargetAddr   = "localhost:8080" 
	Concurrencia = 1000 
	WaitTime     = 5  
)

func main() {
	var wg sync.WaitGroup
	startTest := time.Now()

	fmt.Printf("Iniciando stress test al proxy: %d conexiones...\n", Concurrencia)

	for i := 0; i < Concurrencia; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			startConn := time.Now()
			
			conn, err := net.DialTimeout("tcp", TargetAddr, 5*time.Second)
			if err != nil {
				return 
			}
			defer conn.Close()

			header := make([]byte, 11)
			header[0] = 0x01
			header[1] = 0x00
			header[2] = 0x03 
			binary.BigEndian.PutUint32(header[3:7], 0)
			binary.BigEndian.PutUint32(header[7:11], 0)

			if _, err := conn.Write(header); err != nil {
				return
			}

			resp := make([]byte, 1)
			_, err = conn.Read(resp)
			
			duracion := time.Since(startConn)

			if err == nil {
				if id%50 == 0 {
					fmt.Printf("[Conexión %d] Latencia de handshake: %v\n", id, duracion)
				}
				// Mantenemos abierto para el benchmark de File Descriptors
				time.Sleep(time.Duration(WaitTime) * time.Second)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("\n--- Benchmark finalizado en %v ---\n", time.Since(startTest))
}
