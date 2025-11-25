package main

import (
	"flag"
	"fmt"
	"time"
	
	// Import core
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/p2p"

	{{range .Imports}}	"{{.}}"
	{{end}}
)

var (
	BuiltInNodeName   string = "Unknown Node"
	DefaultConfigPath string = ""
)

func main() {
	configPath := flag.String("config", DefaultConfigPath, "Đường dẫn file cấu hình")
	flag.Parse()
	if *configPath == "" {
		fmt.Println("❌ Vui lòng nhập file config hoặc build lại kèm config mặc định.")
		fmt.Println("VD: ./node -config my_config.yaml")
		return
	}

	conf, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Không đọc được file config tại: %s\n", *configPath)
		panic(err)
	}
	// Khởi tạo Core
	db := database.InitDB(conf.DBPath)
	defer db.Close()
	chain := core.InitBlockchain(db)
	contractManager := contract.NewManager(chain)

	fmt.Println("📦 Đang nạp Chaincode riêng cho Node này...")
{{range .Registrations}}	contractManager.RegisterApp({{.}})
{{end}}

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