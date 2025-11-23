package main

import (
	"embed"
	"flag"
	"fmt"
	"khoai-chain/pkg/cli" // Import package cli vừa tạo
	"os"
)

//go:embed cmd internal pkg examples go.mod go.sum
var sourceCode embed.FS

// ... (Các hàm support cũ giữ nguyên: NodeConfig, prepareDocker...)

func main() {
	app := cli.NewCLI()

	// --- LỆNH 1: BUILD (Lệnh chính) ---
	app.AddCommand("build b", "Build blockchain node (Default)", func(args []string) error {
		// 1. Định nghĩa cờ RIÊNG cho lệnh build
		buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
		configFile := buildCmd.String("input", "", "Path to config file")

		// Hỗ trợ flag mảng cho -addchaincode
		var chaincodePaths stringArray
		buildCmd.Var(&chaincodePaths, "addchaincode", "Path to chaincode")

		// 2. Parse tham số
		if err := buildCmd.Parse(args); err != nil {
			return err
		}

		if *configFile == "" {
			return fmt.Errorf("missing input config. Usage: -input <path>")
		}

		// 3. Gọi logic xử lý chính (Hàm cũ của bạn)
		// Lưu ý: Sửa lại logic cũ để nhận tham số thay vì tự parse flag
		runBuildProcess(*configFile, chaincodePaths)
		return nil
	})

	// --- LỆNH 2: HELP ---
	app.AddCommand("help h", "Show help message", func(args []string) error {
		app.PrintHelp()
		return nil
	})

	// --- LỆNH 3: VERSION ---
	app.AddCommand("version v", "Show version info", func(args []string) error {
		fmt.Println("Khoai Builder v1.0.0 - Docker Edition 🐳")
		return nil
	})

	// --- LỆNH 4: CLEAN ---
	app.AddCommand("clean c", "Remove build artifacts", func(args []string) error {
		fmt.Println("🧹 Cleaning up build folder...")
		os.RemoveAll("build")
		fmt.Println("✅ Done.")
		return nil
	})

	// KÍCH HOẠT
	app.Run()
}

// --- HÀM LOGIC CŨ (Đã tách ra để gọn code) ---
func runBuildProcess(configFile string, chaincodePaths []string) {
	fmt.Printf("🚀 Starting build with config: %s\n", configFile)
	if len(chaincodePaths) > 0 {
		fmt.Printf("🧩 Adding chaincodes: %v\n", chaincodePaths)
	}

	// ... [PASTE TOÀN BỘ LOGIC XỬ LÝ DOCKER/EMBED CŨ VÀO ĐÂY] ...
	// (Chỉ cần đổi đoạn flag.Parse() cũ thành dùng biến truyền vào là xong)
}

// Type hỗ trợ flag mảng (như bài trước)
type stringArray []string

func (i *stringArray) String() string { return fmt.Sprint(*i) }
func (i *stringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}
