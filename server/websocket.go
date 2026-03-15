package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// upgradeWebSocket performs a raw RFC 6455 WebSocket handshake and registers
// the client with the hub. No external dependencies.
func upgradeWebSocket(w http.ResponseWriter, r *http.Request, hub *wsHub) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "not a websocket request", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	accept := computeAcceptKey(key)

	hijack, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hijack.Hijack()
	if err != nil {
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return
	}

	// Write 101 Switching Protocols
	fmt.Fprintf(bufrw,
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n",
		accept,
	)
	bufrw.Flush()

	client := &rawWSClient{conn: conn, bufrw: bufrw, sendCh: make(chan []byte, 64)}

	hub.mu.Lock()
	hub.clients[&wsClient{conn: client, send: client.sendCh}] = true
	hub.mu.Unlock()

	wsClientPtr := func() *wsClient {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		for c := range hub.clients {
			if c.conn == client {
				return c
			}
		}
		return nil
	}()

	// Write pump
	go func() {
		defer func() {
			hub.mu.Lock()
			if wsClientPtr != nil {
				delete(hub.clients, wsClientPtr)
			}
			hub.mu.Unlock()
			conn.Close()
		}()
		for msg := range client.sendCh {
			if err := client.writeTextFrame(msg); err != nil {
				log.Printf("ws write error: %v", err)
				return
			}
		}
	}()

	// Read pump (drain pings / handle close)
	go func() {
		defer conn.Close()
		for {
			_, err := readFrame(bufrw.Reader)
			if err != nil {
				return
			}
		}
	}()
}

func computeAcceptKey(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

type rawWSClient struct {
	conn   net.Conn
	bufrw  *bufio.ReadWriter
	sendCh chan []byte
}

func (c *rawWSClient) WriteMessage(_ int, data []byte) error {
	return c.writeTextFrame(data)
}

func (c *rawWSClient) writeTextFrame(data []byte) error {
	// Text frame (opcode 0x01), no masking (server → client)
	n := len(data)
	var header []byte
	header = append(header, 0x81) // FIN + text opcode
	if n < 126 {
		header = append(header, byte(n))
	} else if n < 65536 {
		header = append(header, 126, byte(n>>8), byte(n))
	} else {
		header = append(header, 127,
			0, 0, 0, 0,
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n),
		)
	}
	c.bufrw.Write(header)
	c.bufrw.Write(data)
	return c.bufrw.Flush()
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	// Read 2-byte header
	hdr := make([]byte, 2)
	if _, err := r.Read(hdr); err != nil {
		return nil, err
	}
	masked := hdr[1]&0x80 != 0
	payloadLen := int(hdr[1] & 0x7F)
	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		r.Read(ext)
		payloadLen = int(ext[0])<<8 | int(ext[1])
	case 127:
		ext := make([]byte, 8)
		r.Read(ext)
		payloadLen = int(ext[4])<<24 | int(ext[5])<<16 | int(ext[6])<<8 | int(ext[7])
	}
	var mask [4]byte
	if masked {
		r.Read(mask[:])
	}
	payload := make([]byte, payloadLen)
	r.Read(payload)
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, nil
}
