package main

import (
	"flag"
	"fmt"
	"time"
	"os"
	"path/filepath"
	"strings"
	
	// Import core
	"khoai-chain/pkg/cli"
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
)

func main() {
	// 1. Setup việc tìm file Config (Logic tìm cạnh file exe)
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)
	defaultConfigPath := filepath.Join(exeDir, "config.yaml")

	// Parse Flag để lấy đường dẫn config
	configPathFlag := flag.String("config", defaultConfigPath, "Đường dẫn file cấu hình")
	flag.Parse()

	// 2. Khởi tạo CLI
	nodeCLI := cli.NewCLI()

	// --- COMMAND: RUN (Dành cho Docker/Production) ---
	// Chế độ này tôn trọng tuyệt đối đường dẫn trong file config (VD: /app/data)
	nodeCLI.AddCommand("run", "Chạy node mode Production (Docker)", func(args []string) error {
		fmt.Println("🐳 Mode: DOCKER / PRODUCTION")
		startNode(*configPathFlag, false)
		return nil
	})

	// --- COMMAND: DEV (Dành cho Lập trình viên test nhanh) ---
	nodeCLI.AddCommand("dev", "Chạy node mode Dev (DB lưu cạnh file exe)", func(args []string) error {
		fmt.Println("🛠️  Mode: DEVELOPMENT")
		startNode(*configPathFlag, true)
		return nil
	})

	nodeCLI.Run()
}

func startNode(configPath string, isDevMode bool) {
	// 1. Load Config
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Không đọc được file config tại: %s\n", configPath)
		// Gợi ý đường dẫn tuyệt đối cho dễ debug
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("   (Đường dẫn tuyệt đối: %s)\n", absPath)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("🏭 KHOAI CHAIN NODE: %s\n", BuiltInNodeName)
	fmt.Printf("📂 Config File: %s\n", configPath)

	// 2. XỬ LÝ ĐƯỜNG DẪN DATABASE (Logic quan trọng nhất ở đây)
	finalDBPath := conf.DBPath

	if isDevMode {
		// LOGIC CHO DEV:
		dbName := filepath.Base(conf.DBPath)

		// Và ép nó nằm cạnh file exe
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		finalDBPath = filepath.Join(exeDir, dbName)

		fmt.Println("🔧 Dev Override: Ép DB về thư mục local")
	} else {
		// LOGIC CHO DOCKER / RUN:
		// Giữ nguyên config. Nếu config là "/app/data" thì dùng đúng như thế.
		fmt.Println("🐳 Docker Mode: Sử dụng đường dẫn DB từ config")
	}

	fmt.Printf("💾 Database Path: %s\n", finalDBPath)
	fmt.Println("========================================")

	// 3. Khởi tạo DB
	db := database.InitDB(finalDBPath)
	// Lưu ý: Trong chế độ server chạy mãi mãi (select{}), defer này chỉ chạy khi tắt app (Ctrl+C)
	defer db.Close()

	// 4. Khởi tạo Blockchain
	chain := core.InitBlockchain(db)

	// 5. Khởi tạo Smart Contract Manager
	contractManager := contract.NewManager(chain)

	fmt.Println("📦 Đang nạp Chaincode riêng cho Node này...")
	{{range .Registrations}}	contractManager.RegisterApp({{.}})
	{{end}}

	fmt.Printf("⛓️  Blockchain Height: %d\n", chain.GetBestHeight())

	// 6. Khởi tạo P2P Server
	srv := p2p.NewServer(conf.Port, contractManager)
	go srv.Start()

	// 7. Kết nối Peers (Sau 2s)
	go func() {
		time.Sleep(2 * time.Second)
		if len(conf.Peers) > 0 {
			fmt.Println("🌐 Danh sách Peers trong config:", conf.Peers)

			for _, peerAddr := range conf.Peers {
				targetAddr := peerAddr

				if isDevMode {
					parts := strings.Split(peerAddr, ":")
					// Đảm bảo đúng định dạng host:port
					if len(parts) == 2 {
						port := parts[1]

						// Tạo địa chỉ mới trỏ về localhost
						targetAddr = fmt.Sprintf("localhost:%s", port)
					}
				}

				// Kết nối tới địa chỉ (đã xử lý hoặc giữ nguyên)
				srv.ConnectToPeer(targetAddr)
			}
		}
	}()

	// 8. Block main thread để server chạy mãi mãi
	select {}
}