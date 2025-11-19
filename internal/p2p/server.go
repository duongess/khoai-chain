package p2p

import (
	"fmt"
	"net"
)

// Chữ "P" trong Port nên viết hoa nếu sau này muốn truy cập từ ngoài,
// nhưng để thường cũng được nếu chỉ dùng trong file này.
type Server struct {
	Port string
}

// SỬA: Đổi newServer -> NewServer (Viết hoa chữ N)
func NewServer(port string) *Server {
	return &Server{
		Port: port,
	}
}

// SỬA: Đổi start -> Start (Viết hoa chữ S)
func (s *Server) Start() {
	// Lưu ý: s.Port giờ đã viết hoa
	listener, err := net.Listen("tcp", ":"+s.Port)

	if err != nil {
		panic(fmt.Sprintf("Lỗi không mở được port %s: %v", s.Port, err))
	}
	fmt.Printf("✅ NODE ĐANG CHẠY! Đang lắng nghe tại địa chỉ: 0.0.0.0:%s\n", s.Port)
	fmt.Println("⏳ Đang chờ ai đó kết nối...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Lỗi kết nối: %v\n", err)
			continue
		}
		fmt.Printf("🎉 Có khách mới ghé thăm từ: %s\n", conn.RemoteAddr().String())
		conn.Close()
	}
}
