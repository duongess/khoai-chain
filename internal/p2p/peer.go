package p2p

import (
	"bufio"
	"fmt"
	"net"
)

type Peer struct {
	Conn net.Conn
}

func NewPeer(conn net.Conn) *Peer {
	return &Peer{
		Conn: conn,
	}
}

func (p *Peer) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
}

func (p *Peer) ReadLoop() {
	defer p.Conn.Close()
	reader := bufio.NewReader(p.Conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Peer %s đã ngắt kết nối\n", p.Conn.RemoteAddr())
			return
		}

		fmt.Printf("📩 Nhận từ [%s]: %s", p.Conn.RemoteAddr(), msg)
	}
}
