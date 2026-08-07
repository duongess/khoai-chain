package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"net"
	"sync"
)

type Server struct {
	Endpoint  string
	Peers     map[string]*Peer
	lock      sync.Mutex
	Contracts *contract.ContractManager
}

func NewServer(endpoint string, contracts *contract.ContractManager) *Server {
	return &Server{
		Endpoint:  endpoint,
		Peers:     make(map[string]*Peer),
		lock:      sync.Mutex{},
		Contracts: contracts,
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", s.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not open port %s: %v", s.Endpoint, err))
	}

	fmt.Printf("Server running on port %s. Waiting for connections...\n", s.Endpoint)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		s.AddPeer(conn)
	}
}

func (s *Server) Stop() {
	fmt.Println("Stopping server and disconnecting all peers...")
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, peer := range s.Peers {
		peer.Conn.Close()
	}
	s.Peers = make(map[string]*Peer)
}

func (s *Server) ConnectToPeer(address string) {
	fmt.Printf("Connecting to %s...\n", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Can't connect to %s: %v\n", address, err)
		return
	}
	s.AddPeer(conn)
	fmt.Println("Sending initial sync request...")
	hashes := s.Contracts.Chain.GetBlockHashes()
	req := GetBlocksRequest{
		Type:   MsgGetChain,
		Hashes: hashes,
	}
	reqBytes, _ := json.Marshal(req)
	conn.Write(append(reqBytes, '\n'))

}

func (s *Server) AddPeer(conn net.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peer := NewPeer(conn, s.Contracts)
	s.Peers[conn.RemoteAddr().String()] = peer

	fmt.Printf("Connection successful: %s. Total Peers: %d\n", conn.RemoteAddr(), len(s.Peers))

	go peer.ReadLoop(s)
}

func (s *Server) DisconnectToPeer(address string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.Peers, address)
	fmt.Printf("Peer %s disconnected. Total Peers: %d\n", address, len(s.Peers))
}

func (s *Server) JoinNetwork(address string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Println("Joining network via", address)
}

func (s *Server) ListPeers() {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Println("Current Peers:")
	for addr := range s.Peers {
		fmt.Printf("- %s\n", addr)
	}
}

func (s *Server) Broadcast(msg string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Printf("Broadcasting message: '%s' to %d nodes...\n", msg, len(s.Peers))

	for _, peer := range s.Peers {
		go func(p *Peer) {
			p.Send([]byte(msg + "\n"))
		}(peer)
	}
}
