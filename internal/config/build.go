package config

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yeka/zip"
	"gopkg.in/yaml.v3"
)

type ConfigContent struct {
	NodeName   string            `yaml:"node_name"`
	Host       string            `yaml:"host"`
	Port       string            `yaml:"port"`
	DBPath     string            `yaml:"db_path"`
	Peers      []string          `yaml:"peers"`
	Chaincodes []ChaincodeConfig `yaml:"chaincodes"` // Nhớ định nghĩa struct ChaincodeConfig
}

func LoadConfig(filePath string) (*ConfigContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy file config: %v", err)
	}

	var conf ConfigContent
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("lỗi định dạng YAML: %v", err)
	}

	return &conf, nil
}

// Thêm tham số fs embed.FS để code linh hoạt hơn
func BuildExe(nodeDir string) error {
	zipFilePath := filepath.Join(nodeDir, "khoai_protected.zip") // Dùng filepath.Join cho an toàn OS

	// 1. Đọc Env lấy Password
	envData, err := sourceCode.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("không đọc được file .env trong embed: %v", err)
	}

	myEnv, err := godotenv.Unmarshal(string(envData))
	if err != nil {
		return err
	}

	password := myEnv["KHOAI_PASS"]
	if password == "" {
		return fmt.Errorf("❌ Lỗi: Chưa set biến môi trường KHOAI_PASS")
	}

	// 2. Xác định thư mục cha (RAM hoặc Disk)
	baseDir := getSecureTempDir()

	// Tạo thư mục tạm cụ thể (Ví dụ: /dev/shm/khoai_build_99381)
	tempDir, err := os.MkdirTemp(baseDir, "khoai_build_*")
	if err != nil {
		return fmt.Errorf("không tạo được thư mục tạm: %v", err)
	}

	// Dọn dẹp thư mục tạm cụ thể này sau khi xong
	defer os.RemoveAll(tempDir)

	if baseDir == "/dev/shm" {
		fmt.Println("🛡️  Mode: Secure RAM Disk (Linux/Docker)")
	} else {
		fmt.Println("💾 Mode: Standard Disk (Windows/Mac)")
	}

	fmt.Println("🔑 Đã lấy được Key, bắt đầu giải nén vào:", tempDir)

	// 3. Giải nén vào tempDir (QUAN TRỌNG: Phải dùng tempDir, KHÔNG dùng baseDir)
	err = unzipWithPassword(zipFilePath, tempDir, password)
	if err != nil {
		return fmt.Errorf("sai mật khẩu hoặc lỗi zip: %v", err)
	}

	fmt.Println("📂 Giải nén xong. Bắt đầu build exe...")

	// 4. Output file exe
	absNodeDir, err := filepath.Abs(nodeDir)
	if err != nil {
		return fmt.Errorf("lỗi lấy đường dẫn tuyệt đối: %v", err)
	}

	outputName := "khoai-node.exe"
	if runtime.GOOS != "windows" {
		outputName = "khoai-node"
	}
	outputExe := filepath.Join(absNodeDir, outputName)

	fmt.Println("Tạo ra file exe ở: ", outputExe)

	// 5. Build
	cmd := exec.Command("go", "build", "-o", outputExe, "./cmd/node")

	// QUAN TRỌNG: Trỏ lệnh build vào đúng thư mục chứa source code vừa giải nén
	cmd.Dir = tempDir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Build thất bại: %v", err)
	}

	fmt.Println("✅ Build thành công!")
	return nil
}

// ... (Hàm unzipWithPassword và getSecureTempDir giữ nguyên như cũ là OK)
func unzipWithPassword(src, dest, password string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		if f.IsEncrypted() {
			f.SetPassword(password)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func getSecureTempDir() string {
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/dev/shm"); err == nil {
			return "/dev/shm"
		}
	}
	return ""
}
