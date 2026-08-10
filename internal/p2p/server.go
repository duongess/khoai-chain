package p2p

import (
	"bufio"
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
		// An inbound socket is not a member yet. In particular, receiving a
		// JOIN_NETWORK request must not add the requester before approval.
		go NewPeer(conn, s.Contracts).ReadLoop(s)
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
	if address == "" || address == s.Endpoint || s.HasPeer(address) {
		return
	}
	fmt.Printf("Connecting to %s...\n", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Can't connect to %s: %v\n", address, err)
		return
	}
	s.addPeer(conn, address)
	hello, _ := json.Marshal(PeerListMessage{Type: MsgPeerList, Sender: s.Endpoint})
	_, _ = conn.Write(append(hello, '\n'))
	fmt.Println("Sending initial sync request...")
	if s.Contracts == nil || s.Contracts.Chain == nil {
		return
	}
	hashes := s.Contracts.Chain.GetBlockHashes()
	req := GetBlocksRequest{
		Type:   MsgGetChain,
		Hashes: hashes,
	}
	reqBytes, _ := json.Marshal(req)
	conn.Write(append(reqBytes, '\n'))

}

func (s *Server) AddPeer(conn net.Conn) {
	s.addPeer(conn, conn.RemoteAddr().String())
}

func (s *Server) addPeer(conn net.Conn, endpoint string) {
	peer := NewPeer(conn, s.Contracts)
	peer.Endpoint = endpoint
	s.addPeerObject(peer)
}

func (s *Server) addPeerObject(peer *Peer) {
	s.registerPeer(peer)
	go peer.ReadLoop(s)
}

// RegisterPendingPeer promotes a socket that completed the JOIN/ACCEPT flow.
// It is intentionally a Server operation, not a wire-protocol operation.
func (s *Server) RegisterPendingPeer(peer *Peer) {
	s.registerPeer(peer)
}

func (s *Server) registerPeer(peer *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	endpoint := peer.Endpoint
	if endpoint == "" {
		endpoint = peer.Conn.RemoteAddr().String()
		peer.Endpoint = endpoint
	}
	if existing, ok := s.Peers[endpoint]; ok && existing != peer {
		// Prefer the existing connection and avoid duplicate mesh links.
		go peer.Conn.Close()
		return
	}
	s.Peers[endpoint] = peer
	peer.registered = true

	fmt.Printf("Connection successful: %s. Total Peers: %d\n", endpoint, len(s.Peers))
}

func (s *Server) RemovePeer(p *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	addr := p.Endpoint
	if addr == "" {
		addr = p.Conn.RemoteAddr().String()
	}
	if existing, ok := s.Peers[addr]; ok && existing == p {
		delete(s.Peers, addr)
		fmt.Printf("Peer %s removed from list. Total Peers: %d\n", addr, len(s.Peers))
	}
}

func (s *Server) DisconnectToPeer(address string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Find the peer and close its connection. This will trigger its ReadLoop to exit
	// and call RemovePeer via its defer statement.
	if peer, ok := s.Peers[address]; ok {
		fmt.Printf("Closing connection to peer %s...\n", address)
		// Closing the connection will cause the peer's ReadLoop to error out,
		// which in turn triggers the deferred RemovePeer call.
		peer.Conn.Close()
	} else {
		fmt.Printf("Peer %s not found for disconnection.\n", address)
	}
}

func (s *Server) JoinNetwork(address string) {
	if address == "" || address == s.Endpoint || s.HasPeer(address) {
		return
	}

	fmt.Println("Joining network via", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Can't join network via %s: %v\n", address, err)
		return
	}

	join, err := json.Marshal(JoinNetworkRequest{Type: MsgJoinNetwork, Address: s.Endpoint})
	if err != nil {
		conn.Close()
		return
	}
	if _, err := conn.Write(append(join, '\n')); err != nil {
		conn.Close()
		return
	}

	joiningPeer := NewPeer(conn, s.Contracts)
	reader := bufio.NewReader(conn)
	joiningPeer.reader = reader
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Printf("Join request to %s failed: %v\n", address, err)
		conn.Close()
		return
	}
	var accepted AcceptJoinMessage
	if err := json.Unmarshal(line, &accepted); err != nil || accepted.Type != MsgAcceptJoin {
		fmt.Printf("Join request to %s was not accepted\n", address)
		conn.Close()
		return
	}

	// The accepted JOIN socket is the actual peer connection. It is registered
	// only after ACCEPT_JOIN, while the peer-list response is read by ReadLoop.
	joiningPeer.Endpoint = address
	s.addPeerObject(joiningPeer)
}

func (s *Server) LeaveNetwork() {
	leave, _ := json.Marshal(LeaveNetworkMessage{Type: MsgLeaveNetwork, Address: s.Endpoint})
	for _, peer := range s.peerSnapshot() {
		_ = peer.Send(append(leave, '\n'))
	}
	s.Stop()
}

func (s *Server) RemovePeerByEndpoint(endpoint string) {
	s.lock.Lock()
	peer, ok := s.Peers[endpoint]
	if ok {
		delete(s.Peers, endpoint)
	}
	s.lock.Unlock()
	if ok {
		_ = peer.Conn.Close()
		fmt.Printf("Peer %s left the network. Total Peers: %d\n", endpoint, len(s.GetPeerList()))
	}
}

func (s *Server) HasPeer(endpoint string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.Peers[endpoint]
	return ok
}

func (s *Server) peerSnapshot() []*Peer {
	s.lock.Lock()
	defer s.lock.Unlock()
	peers := make([]*Peer, 0, len(s.Peers))
	for _, peer := range s.Peers {
		peers = append(peers, peer)
	}
	return peers
}

func (s *Server) ListPeers() {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Println("Current Peers:")
	for addr := range s.Peers {
		fmt.Printf("- %s\n", addr)
	}
}

func (s *Server) GetPeerList() []string {
	s.lock.Lock()
	defer s.lock.Unlock()

	peers := make([]string, 0, len(s.Peers))
	for addr := range s.Peers {
		peers = append(peers, addr)
	}

	return peers
}

func (s *Server) Broadcast(msg string) {
	peers := s.peerSnapshot()
	fmt.Printf("Broadcasting message: '%s' to %d nodes...\n", msg, len(peers))

	for _, peer := range peers {
		go func(p *Peer) {
			p.Send([]byte(msg + "\n"))
		}(peer)
	}
}
