package main

import (
	"flag"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/internal/p2p"
	"time"
)

// Các biến này sẽ được Builder điền vào lúc Build
var (
	BuiltInNodeName   string = "Unknown Node"
	DefaultConfigPath string = "" // <--- THÊM BIẾN NÀY
)

func main() {
	// 1. Flag vẫn giữ để ai thích thì override, nhưng giá trị mặc định lấy từ biến toàn cục
	configPath := flag.String("config", DefaultConfigPath, "Đường dẫn file cấu hình")
	flag.Parse()

	// [QUAN TRỌNG] Nếu không nhập flag và cũng không có default -> Báo lỗi
	if *configPath == "" {
		fmt.Println("❌ Vui lòng nhập file config hoặc build lại kèm config mặc định.")
		fmt.Println("VD: ./node -config my_config.yaml")
		return
	}

	// 2. Load Config
	// Lưu ý: Nếu DefaultConfigPath được tiêm vào, nó sẽ tự load file đó
	conf, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Không đọc được file config tại: %s\n", *configPath)
		panic(err)
	}

	// ... (Phần còn lại giữ nguyên như cũ)
	fmt.Println("========================================")
	fmt.Printf("🏭 KHOAI CHAIN NODE: %s\n", BuiltInNodeName)
	fmt.Printf("📂 Config File: %s\n", *configPath) // In ra để kiểm tra
	fmt.Println("========================================")

	// db := database.InitDB(conf.DBPath)
	// defer db.Close()

	srv := p2p.NewServer(conf.Port)
	go srv.Start()

	go func() {
		time.Sleep(2 * time.Second)
		for _, peerAddr := range conf.Peers {
			srv.ConnectToPeer(peerAddr)
		}
	}()

	select {}
}
