package main

import (
	"flag"
	"khoai-chain/internal/p2p" // Đảm bảo đường dẫn này đúng với go.mod của bạn
)

func main() {
	port := flag.String("port", "8000", "Cổng mạng để lắng nghe")
	flag.Parse()

	// Bây giờ main mới nhìn thấy NewServer
	srv := p2p.NewServer(*port)

	// SỬA: start -> Start (Viết hoa cho khớp với bên kia)
	srv.Start()
}
