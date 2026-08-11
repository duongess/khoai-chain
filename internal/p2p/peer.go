package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"net"
	"sync"
)

type Peer struct {
	Conn       net.Conn
	Contracts  *contract.ContractManager
	Endpoint   string
	joinPeers  []string
	registered bool
	reader     *bufio.Reader
	writeLock  sync.Mutex
}

func NewPeer(conn net.Conn, contracts *contract.ContractManager) *Peer {
	return &Peer{
		Conn:      conn,
		Contracts: contracts,
	}
}

func (p *Peer) Send(data []byte) error {
	p.writeLock.Lock()
	defer p.writeLock.Unlock()
	_, err := p.Conn.Write(data)
	return err
}

func (p *Peer) ReadLoop(s *Server) {
	// When the loop exits (due to error or disconnection),
	// remove the peer from the server's list and close the connection.
	defer func() {
		if p.registered {
			s.RemovePeer(p)
		} else {
			s.RemovePendingPeer(p)
		}
		p.Conn.Close()
	}()
	reader := p.reader
	if reader == nil {
		reader = bufio.NewReader(p.Conn)
		p.reader = reader
	}

	for { // 1. Read message
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Peer %s disconnected: %v\n", p.Conn.RemoteAddr(), err)
			return
		}

		// 2. Call Handler
		responseBytes, err := handleMessage(msg, s, p.Contracts, p)

		if err != nil {
			fmt.Println("Error handling message:", err)
		}

		// 3. If Handler returns data -> Send it
		if len(responseBytes) > 0 {
			// Ensure newline
			if responseBytes[len(responseBytes)-1] != '\n' {
				responseBytes = append(responseBytes, '\n')
			}
			if err := p.Send(responseBytes); err != nil {
				return
			}
		}

		if p.joinPeers != nil {
			peerList, err := json.Marshal(PeerListMessage{Type: MsgPeerList, Sender: s.Endpoint, Peers: p.joinPeers})
			if err != nil || p.Send(append(peerList, '\n')) != nil {
				return
			}
			p.joinPeers = nil
			s.RegisterPendingPeer(p)
		}
	}
}
