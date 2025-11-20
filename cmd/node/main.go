package main

import (
	"flag"
	"fmt"
	"khoai-chain/internal/p2p"
	"time"
)

func main() {
	port := flag.String("port", "8000", "Port lắng nghe")
	target := flag.String("connect", "", "IP node khác")
	flag.Parse()

	srv := p2p.NewServer(*port)
	go srv.Start()

	// Nếu có target thì kết nối
	if *target != "" {
		time.Sleep(1 * time.Second)
		srv.ConnectToPeer(*target)
	}

	// [TEST PHÂN TÁN]
	// Tạo một vòng lặp gửi tin nhắn định kỳ để xem các node có nhận được không
	go func() {
		for {
			time.Sleep(5 * time.Second) // Cứ 5 giây gửi 1 lần
			if len(srv.Peers) > 0 {
				msg := fmt.Sprintf("Chào, tôi là Node %s", *port)
				srv.Broadcast(msg)
			}
		}
	}()

	select {}
}
