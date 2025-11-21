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

	for {
		msg, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			fmt.Printf("❌ Peer %s đã ngắt kết nối\n", p.Conn.RemoteAddr())
			return
		}

		fmt.Printf("📩 Nhận từ [%s]: %s", p.Conn.RemoteAddr(), msg)
		result, err := HandleMessage(msg, p.Contracts)
		if err != nil {
			fmt.Println(err)
			errorMsg := fmt.Sprintf(`{"status":"error", "message":"%v"}`+"\n", err)
			p.Send([]byte(errorMsg))
			return
		}

		if len(result) > 0 {
			if result[len(result)-1] != '\n' {
				result = append(result, '\n')
			}
		} else {
			result = append(result, '\n')
		}
		p.Send(result)
	}
}
