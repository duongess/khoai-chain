package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Cấu trúc chỉ để đọc cái node_name từ file config
type NodeConfig struct {
	NodeName string `yaml:"node_name"`
}

func main() {
	// 1. Nhận tham số là file config đầu vào
	configFile := flag.String("input", "", "Đường dẫn file config (VD: configs/node_contractor.yaml)")
	flag.Parse()

	if *configFile == "" {
		log.Fatal("❌ Vui lòng nhập file config: go run cmd/builder/main.go -input configs/node_contractor.yaml")
	}

	// 2. Đọc file Config
	fmt.Printf("📖 Đang đọc cấu hình từ: %s\n", *configFile)
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal("❌ Lỗi đọc file:", err)
	}

	var conf NodeConfig
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal("❌ Lỗi parse YAML:", err)
	}

	// 3. Xác định tên file output (VD: TongThau_Coteccons.exe)
	outputName := conf.NodeName
	if runtime.GOOS == "windows" {
		outputName += ".exe"
	}

	// Tạo thư mục build nếu chưa có
	if _, err := os.Stat("build"); os.IsNotExist(err) {
		os.Mkdir("build", 0755)
	}
	outputPath := filepath.Join("build", outputName)

	// 4. Chuẩn bị lệnh Build thần thánh
	// -ldflags "-X main.BuiltInNodeName=..." chính là kỹ thuật tiêm biến
	ldflags := fmt.Sprintf(
		"-X 'main.BuiltInNodeName=%s' -X 'main.DefaultConfigPath=%s'",
		conf.NodeName,
		*configFile, // Tiêm chính cái đường dẫn file config đầu vào vào trong exe
	)

	fmt.Printf("🔨 Đang đúc Node: %s (Config mặc định: %s)...\n", outputName, *configFile)

	cmd := exec.Command("go", "build",
		"-o", outputPath,
		"-ldflags", ldflags,
		"cmd/node/main.go",
	)

	// Gắn log ra màn hình để xem tiến độ
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 5. Chạy lệnh
	err = cmd.Run()
	if err != nil {
		log.Fatal("❌ Build thất bại:", err)
	}

	fmt.Println("✅ XONG! File chạy nằm tại:", outputPath)
	fmt.Println("👉 Bạn có thể gửi file này cho:", conf.NodeName)
}
