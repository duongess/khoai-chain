package p2p

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	Port  string
	Peers []*Peer
	lock  sync.Mutex
}

func NewServer(port string) *Server {
	return &Server{
		Port:  port,
		Peers: []*Peer{},
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		panic(fmt.Sprintf("❌ Không thể mở port %s: %v", s.Port, err))
	}

	fmt.Printf("✅ Server đang chạy port %s. Chờ kết nối...\n", s.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️ Lỗi Accept: %v\n", err)
			continue
		}
		s.AddPeer(conn)
	}
}

func (s *Server) ConnectToPeer(address string) {
	fmt.Printf("🔌 Đang kết nối đến %s...\n", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("❌ Không thể kết nối đến %s: %v\n", address, err)
		return
	}

	s.AddPeer(conn)
}

func (s *Server) AddPeer(conn net.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peer := NewPeer(conn)
	s.Peers = append(s.Peers, peer)

	fmt.Printf("🤝 Kết nối thành công: %s. Tổng số Peer: %d\n", conn.RemoteAddr(), len(s.Peers))

	go peer.ReadLoop()
}

func (s *Server) Broadcast(msg string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Printf("📢 Đang lan truyền tin: '%s' tới %d nodes...\n", msg, len(s.Peers))

	for _, peer := range s.Peers {
		go func(p *Peer) {
			p.Send([]byte(msg + "\n"))
		}(peer)
	}
}
