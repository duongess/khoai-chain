package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

//go:embed cmd internal pkg examples go.mod go.sum
var sourceCode embed.FS

type NodeConfig struct {
	NodeName string `yaml:"node_name"`
}

func main() {
	// 1. Nhận input
	configFile := flag.String("input", "", "File config")
	flag.Parse()
	if *configFile == "" {
		log.Fatal("❌ Thiếu file config: ./khoai -input configs/node_A.yaml")
	}

	// 2. Đọc Config (Lấy đường dẫn tuyệt đối)
	absConfigPath, _ := filepath.Abs(*configFile)
	configData, err := os.ReadFile(absConfigPath)
	if err != nil {
		log.Fatal("❌ Lỗi đọc config:", err)
	}
	var conf NodeConfig
	yaml.Unmarshal(configData, &conf)

	// 3. TẠO "NHÀ MÁY" TẠM THỜI (Temp Dir)
	tempDir, err := os.MkdirTemp("", "khoai_build_env_*")
	if err != nil {
		log.Fatal("❌ Lỗi tạo temp dir:", err)
	}
	defer os.RemoveAll(tempDir) // Xây xong thì đập đi cho sạch

	fmt.Println("📦 Đang giải nén source code lõi...")

	// 4. BUNG FILE TỪ BỤNG EXE RA THƯ MỤC TẠM
	fs.WalkDir(sourceCode, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join(tempDir, path)
		if d.IsDir() {
			os.MkdirAll(destPath, 0755)
			return nil
		}
		// Đọc file từ bộ nhớ embed
		content, _ := sourceCode.ReadFile(path)
		// Ghi ra ổ cứng ảo
		return os.WriteFile(destPath, content, 0644)
	})

	// 5. TIẾN HÀNH BUILD
	cwd, _ := os.Getwd()
	outputName := conf.NodeName
	if runtime.GOOS == "windows" {
		outputName += ".exe"
	}

	os.MkdirAll(filepath.Join(cwd, "build"), 0755)
	finalOutput := filepath.Join(cwd, "build", outputName)

	fmt.Printf("🔨 Đang đúc Node: %s...\n", outputName)

	ldflags := fmt.Sprintf("-X 'main.BuiltInNodeName=%s' -X 'main.DefaultConfigPath=%s'", conf.NodeName, absConfigPath)

	// Gọi lệnh go build TRONG thư mục tạm
	cmd := exec.Command("go", "build", "-o", finalOutput, "-ldflags", ldflags, "cmd/node/main.go")
	cmd.Dir = tempDir // <--- Quan trọng: Chuyển chỗ làm việc vào tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // Giữ nguyên môi trường (GOPATH, v.v.)

	if err := cmd.Run(); err != nil {
		log.Fatal("❌ Build thất bại:", err)
	}

	fmt.Println("✅ XONG! File chạy tại:", finalOutput)
}
