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
	Chaincodes []ChaincodeConfig `yaml:"chaincodes"` // Remember to define the ChaincodeConfig struct
}

func LoadConfig(filePath string) (*ConfigContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %v", err)
	}

	var conf ConfigContent
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("YAML format error: %v", err)
	}

	return &conf, nil
}

// Add fs embed.FS parameter for more flexible code
func BuildExe(nodeDir string) error {
	zipFilePath := filepath.Join(nodeDir, "khoai_protected.zip") // Use filepath.Join for OS safety

	// 1. Read Env to get Password
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
		return fmt.Errorf("❌ Error: KHOAI_PASS environment variable not set")
	}

	// 2. Determine parent directory (RAM or Disk)
	baseDir := getSecureTempDir()

	// Create a specific temporary directory (e.g., /dev/shm/khoai_build_99381)
	tempDir, err := os.MkdirTemp(baseDir, "khoai_build_*")
	if err != nil {
		return fmt.Errorf("không tạo được thư mục tạm: %v", err)
	}

	// Clean up this specific temporary directory when done
	defer os.RemoveAll(tempDir)

	if baseDir == "/dev/shm" {
		fmt.Println("🛡️  Mode: Secure RAM Disk (Linux/Docker)")
	} else {
		fmt.Println("💾 Mode: Standard Disk (Windows/Mac)")
	}

	fmt.Println("🔑 Key retrieved, starting extraction to:", tempDir)

	// 3. Unzip into tempDir (IMPORTANT: Must use tempDir, NOT baseDir)
	err = unzipWithPassword(zipFilePath, tempDir, password)
	if err != nil {
		return fmt.Errorf("sai mật khẩu hoặc lỗi zip: %v", err)
	}

	fmt.Println("📂 Unzip complete. Starting exe build...")

	// 4. Output exe file
	absNodeDir, err := filepath.Abs(nodeDir)
	if err != nil {
		return fmt.Errorf("lỗi lấy đường dẫn tuyệt đối: %v", err)
	}

	outputName := "khoai-node.exe"
	if runtime.GOOS != "windows" {
		outputName = "khoai-node"
	}
	outputExe := filepath.Join(absNodeDir, outputName)

	fmt.Println("Creating exe file at: ", outputExe)

	// 5. Build
	cmd := exec.Command("go", "build", "-o", outputExe, "./cmd/node")

	// IMPORTANT: Point the build command to the directory containing the just-extracted source code
	cmd.Dir = tempDir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Build failed: %v", err)
	}

	fmt.Println("✅ Build successful!")
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
