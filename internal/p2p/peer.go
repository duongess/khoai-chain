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
		// 1. Đọc tin nhắn
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("❌ Peer %s đã ngắt kết nối\n", p.Conn.RemoteAddr())
			return
		}

		// 2. Gọi Handler
		responseBytes, err := HandleMessage(msg, p.Contracts)

		if err != nil {
			fmt.Println("Lỗi xử lý:", err)
		}

		// 3. Nếu Handler trả về dữ liệu -> Gửi đi
		if len(responseBytes) > 0 {
			// Đảm bảo có xuống dòng
			if responseBytes[len(responseBytes)-1] != '\n' {
				responseBytes = append(responseBytes, '\n')
			}
			p.Send(responseBytes)
		}
	}
}
