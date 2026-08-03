package p2p

import (
	"bufio"
	"fmt"
	"khoai-chain/internal/contract"
	"net"
)

type Peer struct {
	Conn      net.Conn
	Contracts *contract.ContractManager
}

func NewPeer(conn net.Conn, contracts *contract.ContractManager) *Peer {
	return &Peer{
		Conn:      conn,
		Contracts: contracts,
	}
}

func (p *Peer) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
}

func (p *Peer) ReadLoop() {
	defer p.Conn.Close()
	reader := bufio.NewReader(p.Conn)

	for { // 1. Read message
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Peer %s disconnected\n", p.Conn.RemoteAddr())
			return
		}

		// 2. Call Handler
		responseBytes, err := HandleMessage(msg, p.Contracts)

		if err != nil {
			fmt.Println("Error handling message:", err)
		}

		// 3. If Handler returns data -> Send it
		if len(responseBytes) > 0 {
			// Ensure newline
			if responseBytes[len(responseBytes)-1] != '\n' {
				responseBytes = append(responseBytes, '\n')
			}
			p.Send(responseBytes)
		}
	}
}
