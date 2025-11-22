package main

import (
	"flag"
	"fmt"
	"khoai-chain/examples"
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
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

	// 1. Khởi tạo DB (Kho chứa)
	db := database.InitDB(conf.DBPath)
	defer db.Close()

	// 2. Khởi tạo Blockchain (Bộ não quản lý)
	// Truyền con trỏ DB vào để Blockchain dùng
	chain := core.InitBlockchain(db)

	contractManager := contract.NewManager(chain)

	contractManager.RegisterApp(examples.NewUsageExamples())
	// In ra độ cao hiện tại để kiểm tra
	fmt.Printf("⛓️  Blockchain Height: %d\n", chain.GetBestHeight())

	srv := p2p.NewServer(conf.Port, contractManager)
	go srv.Start()

	go func() {
		time.Sleep(2 * time.Second)
		for _, peerAddr := range conf.Peers {
			srv.ConnectToPeer(peerAddr)
		}
	}()

	select {}
}
