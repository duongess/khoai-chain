package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"net"
	"sync"
	"time"
)

const joinRequestTTL = 5 * time.Minute

// JoinRequest is deliberately RAM-only. A request becomes a persistent peer
// only after an explicit ACCEPT_JOIN.
type JoinRequest struct {
	RequestID   string
	NodeID      string
	Endpoint    string // P2P endpoint advertised to other nodes.
	APIEndpoint string // HTTP endpoint advertised inside Docker.
	ExpiresAt   time.Time
	outbound    bool
}

type Server struct {
	Endpoint           string
	P2PListenEndpoint  string
	HTTPListenEndpoint string
	HTTPEndpoint       string
	NodeID             string
	Peers              map[string]*Peer
	PendingRequests    map[string]*JoinRequest
	lock               sync.RWMutex
	Contracts          *contract.ContractManager
	config             *config.ConfigContent
	configPath         string
}

func NewServer(endpoint string, contracts *contract.ContractManager) *Server {
	return &Server{
		Endpoint:        endpoint,
		Peers:           make(map[string]*Peer),
		PendingRequests: make(map[string]*JoinRequest),
		Contracts:       contracts,
	}
}

// ConfigurePersistence attaches this server to its runtime config.yaml. It is
// called by the node executable after loading config; join requests are never
// stored here, only accepted peers are.
func (s *Server) ConfigurePersistence(conf *config.ConfigContent, configPath string) {
	s.lock.Lock()
	s.config = conf
	s.configPath = configPath
	if conf != nil {
		s.NodeID = conf.NodeID
		if conf.P2PEndpoint != "" {
			s.Endpoint = conf.P2PEndpoint
		}
		if conf.P2PListenEndpoint != "" {
			s.P2PListenEndpoint = conf.P2PListenEndpoint
		}
		if conf.HTTPListenEndpoint != "" {
			s.HTTPListenEndpoint = conf.HTTPListenEndpoint
		}
		if conf.HTTPEndpoint != "" {
			s.HTTPEndpoint = conf.HTTPEndpoint
		}
	}
	s.lock.Unlock()
}

func (s *Server) Start() {
	listenEndpoint := s.P2PListenEndpoint
	if listenEndpoint == "" {
		listenEndpoint = s.Endpoint
	}
	listener, err := net.Listen("tcp", listenEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not open port %s: %v", listenEndpoint, err))
	}

	fmt.Printf("P2P data plane listening on %s (advertised as %s)\n", listenEndpoint, s.Endpoint)
	go s.StartPeerAPI()
	go s.connectPersistedPeers()

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
	fmt.Println("Sending initial sync request...")
	if s.Contracts == nil || s.Contracts.Chain == nil {
		return
	}
	hashes := s.Contracts.Chain.GetBlockHashes()
	req := GetBlocksRequest{
		Type:   MsgGetChain,
		Sender: s.Endpoint,
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

func (s *Server) LeaveNetwork() {
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
		s.removePersistedPeer(endpoint)
		fmt.Printf("Peer %s left the network. Total Peers: %d\n", endpoint, len(s.GetPeerList()))
	}
}

func (s *Server) HasPeer(endpoint string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.Peers[endpoint]
	return ok
}

// IsLocalEndpoint accepts the configured listener and shorthand addresses such
// as :8082 used by the local khoai CLI.
func (s *Server) IsLocalEndpoint(endpoint string) bool {
	if endpoint == s.Endpoint {
		return true
	}
	_, requestedPort, requestErr := net.SplitHostPort(endpoint)
	_, serverPort, serverErr := net.SplitHostPort(s.Endpoint)
	return requestErr == nil && serverErr == nil && requestedPort == serverPort
}

func (s *Server) peerSnapshot() []*Peer {
	s.lock.RLock()
	defer s.lock.RUnlock()
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	peers := make([]string, 0, len(s.Peers))
	for addr := range s.Peers {
		peers = append(peers, addr)
	}

	return peers
}

func (s *Server) addPendingRequest(request *JoinRequest) {
	s.lock.Lock()
	s.PendingRequests[request.RequestID] = request
	s.lock.Unlock()
	time.AfterFunc(time.Until(request.ExpiresAt), func() {
		s.expirePendingRequest(request.RequestID)
	})
}
func (s *Server) expirePendingRequest(requestID string) {
	s.lock.Lock()
	request, ok := s.PendingRequests[requestID]
	if !ok || time.Now().Before(request.ExpiresAt) {
		s.lock.Unlock()
		return
	}
	delete(s.PendingRequests, requestID)
	s.lock.Unlock()
	fmt.Printf("Join request %s from %s expired\n", requestID, request.Endpoint)
}

func (s *Server) connectPersistedPeers() {
	s.lock.RLock()
	if s.config == nil {
		s.lock.RUnlock()
		return
	}
	peers := append([]string(nil), s.config.Peers...)
	s.lock.RUnlock()
	for _, endpoint := range peers {
		go s.ConnectToPeer(endpoint)
	}
}

func (s *Server) persistPeer(endpoint string) {
	if endpoint == "" || endpoint == s.Endpoint {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.config == nil || s.configPath == "" {
		return
	}
	for _, peer := range s.config.Peers {
		if peer == endpoint {
			return
		}
	}
	s.config.Peers = append(s.config.Peers, endpoint)
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		fmt.Printf("Could not persist peer %s: %v\n", endpoint, err)
	}
}

func (s *Server) removePersistedPeer(endpoint string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.config == nil || s.configPath == "" {
		return
	}
	peers := s.config.Peers[:0]
	for _, peer := range s.config.Peers {
		if peer != endpoint {
			peers = append(peers, peer)
		}
	}
	s.config.Peers = peers
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		fmt.Printf("Could not remove persisted peer %s: %v\n", endpoint, err)
	}
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
