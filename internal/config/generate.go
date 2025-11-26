package config

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yeka/zip"
	"gopkg.in/yaml.v3"
)

var sourceCode embed.FS

func SetSourceCode(source embed.FS) {
	sourceCode = source
}

type NetworkConfig struct {
	NetworkName string       `yaml:"network_name"`
	Domain      string       `yaml:"domain"`
	ImageBase   string       `yaml:"image_base"`
	ImageTag    string       `yaml:"image_tag"`
	Registry    string       `yaml:"registry"`
	Nodes       []NodeConfig `yaml:"nodes"`
}

type NodeConfig struct {
	Name       string            `yaml:"name"`
	Port       string            `yaml:"port"`
	DBPath     string            `yaml:"db_path"`
	Peers      []string          `yaml:"peers"`
	Chaincodes []ChaincodeConfig `yaml:"chaincodes"`
}

type MainTemplateData struct {
	Imports       []string
	Registrations []string
}

type ChaincodeConfig struct {
	Name        string `yaml:"name"`
	Package     string `yaml:"package"`
	Constructor string `yaml:"constructor"`
}

// --- LOGIC SINH FILE CONFIG RIÊNG CHO TỪNG NODE ---
func GenerateNodeArtifacts(nodeDir string, node NodeConfig, net NetworkConfig) error {
	// A. Sinh file config.yaml cho node này (để nhét vào Image)
	// Lưu ý: Trong Docker, Host thường bind là 0.0.0.0

	finalConfig := ConfigContent{
		NodeName:   node.Name,
		Host:       "0.0.0.0",
		Port:       node.Port,
		DBPath:     "/app/data",
		Peers:      node.Peers,
		Chaincodes: node.Chaincodes,
	}

	configContent, err := yaml.Marshal(finalConfig)

	if err != nil {
		return fmt.Errorf("lỗi marshal config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	if err := generateMainFile(nodeDir, node.Chaincodes); err != nil {
		return fmt.Errorf("lỗi sinh main.go: %v", err)
	}

	if err := ZipFiles(nodeDir, node.Chaincodes); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(nodeDir, "main.go"))

	// B. Sinh Dockerfile (Multi-stage build)
	dockerfileTmpl := `
# --- Stage 1: Builder ---
FROM {{.ImageBase}} AS builder
WORKDIR /app

# 1. Thiết lập môi trường
ENV GO111MODULE=on
ENV GOWORK=off

# 2. Copy toàn bộ code vào
COPY . .

# 👇 [DEBUG CỰC MẠNH] Kiểm tra xem thư mục internal có gì không?
# Nếu lệnh này in ra "INTERNAL MISSING" hoặc danh sách rỗng -> Nguyên nhân ở đây!
RUN echo "==============================================" && \
    echo "📂 KIỂM TRA FILE GO.MOD:" && cat go.mod && \
    echo "----------------------------------------------" && \
    echo "📂 KIỂM TRA THƯ MỤC INTERNAL:" && ls -F internal/ || echo "❌ KHÔNG TÌM THẤY THƯ MỤC INTERNAL" && \
    echo "=============================================="

# 3. Xóa rác môi trường cũ
RUN rm -f go.work go.work.sum
RUN rm -rf vendor/

# 4. Copy main.go thửa riêng
COPY ./build/{{.NodeName}}/main.go ./cmd/node/main.go

# 5. Cập nhật module
RUN go mod tidy

# 6. Build (Thử build bằng đường dẫn package đầy đủ)
# Lưu ý: Thay "khoai-chain" bằng tên module thật nếu khác
RUN CGO_ENABLED=0 GOOS=linux go build -v -o khoai-node ./cmd/node

# --- Stage 2: Runner ---
FROM alpine:3.19
WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/khoai-node .
COPY ./build/{{.NodeName}}/config.yaml ./config.yaml 

RUN mkdir -p /app/data
EXPOSE {{.Port}}

ENTRYPOINT ["./khoai-node", "-config", "./config.yaml"]
`
	// Dữ liệu template
	data := map[string]interface{}{
		"ImageBase": net.ImageBase,
		"NodeName":  node.Name,
		"Port":      node.Port,
	}

	t, _ := template.New("dockerfile").Parse(dockerfileTmpl)
	f, err := os.Create(filepath.Join(nodeDir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

// --- LOGIC SINH DOCKER-COMPOSE ---
func GenerateDockerCompose(baseDir string, net NetworkConfig) error {
	composeTmpl := `version: "3.9"

networks:
  {{.NetworkName}}:
    driver: bridge

services:
{{range .Nodes}}
  {{.Name}}:
    build:
      context: ../  # Trỏ ra root folder chứa source code
      dockerfile: ./build/{{.Name}}/Dockerfile
    image: {{$.Registry}}/{{.Name}}:{{$.ImageTag}}
    container_name: {{.Name}}
    ports:
      - "{{.Port}}:{{.Port}}"
    volumes:
      - ./data/{{.Name}}:/app/data
    networks:
      - {{$.NetworkName}}
    restart: always
{{end}}
`
	t, _ := template.New("compose").Parse(composeTmpl)
	f, err := os.Create(filepath.Join(baseDir, "docker-compose.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, net)
}

// Generate main file
func generateMainFile(outputDir string, chaincodes []ChaincodeConfig) error {

	mainTmpl := `package main

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
	`

	data := MainTemplateData{}
	seen := make(map[string]bool)

	for _, cc := range chaincodes {
		// Thêm import nếu chưa có
		if !seen[cc.Package] {
			data.Imports = append(data.Imports, cc.Package)
			seen[cc.Package] = true
		}
		// Thêm dòng đăng ký: bds.NewBDSContract()
		// Giả sử package tên là phần cuối của đường dẫn (vd: bds)
		pkgName := filepath.Base(cc.Package)
		regLine := fmt.Sprintf("%s.%s()", pkgName, cc.Constructor)
		data.Registrations = append(data.Registrations, regLine)
	}

	// Ghi file
	t, err := template.New("main").Parse(mainTmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outputDir, "main.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// Nén main.go và các smart contract vào khoai_protected.zip của từng node
func ZipFiles(nodeDir string, chaincodes []ChaincodeConfig) error {
	envData, err := sourceCode.ReadFile(".env")

	if err != nil {
		return err
	}
	myEnv, err := godotenv.Unmarshal(string(envData))
	if err != nil {
		return err
	}

	password := myEnv["KHOAI_PASS"]

	if password == "" {
		return fmt.Errorf("❌ Lỗi: Chưa set biến môi trường KHOAI_PASS")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("không lấy được thư mục hiện tại: %v", err)
	}

	mainPath := filepath.Join(nodeDir, "main.go")
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("không tìm thấy file main.go tại %s: %v", mainPath, err)
	}

	outputPath := filepath.Join(nodeDir, "khoai_protected.zip")
	fmt.Printf("🔒 Đang nén main + smart contracts vào '%s' với mật khẩu...\n", outputPath)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("không thể tạo file zip: %v", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	added := make(map[string]bool)

	// 1) Nén toàn bộ source code embed (trừ main để thay thế)
	if err := fs.WalkDir(sourceCode, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == "." {
			return nil
		}

		archivePath := filepath.ToSlash(strings.TrimPrefix(path, "./"))
		if archivePath == "cmd/node/main.go" {
			return nil // sẽ thay bằng main generate
		}
		if added[archivePath] {
			return nil
		}

		content, err := sourceCode.ReadFile(path)
		if err != nil {
			return err
		}

		if err := addBytesToZip(w, archivePath, content, password); err != nil {
			return err
		}
		added[archivePath] = true
		return nil
	}); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("lỗi trong quá trình nén nguồn embed: %v", err)
	}

	// 2) Thay thế cmd/node/main.go bằng file main generate cho node
	if err := addFileToZip(w, mainPath, "cmd/node/main.go", password); err != nil {
		return fmt.Errorf("lỗi trong quá trình nén %s: %v", mainPath, err)
	}
	added["cmd/node/main.go"] = true

	// 3) Nén thêm các smart contract từ config (tuỳ khách hàng bổ sung)
	seenChaincode := make(map[string]bool)
	for _, cc := range chaincodes {
		if cc.Package == "" {
			continue
		}

		resolvedPath, archiveBase, err := resolveChaincodePath(cc.Package, cwd)
		if err != nil {
			return fmt.Errorf("chaincode '%s': %v", cc.Name, err)
		}

		if seenChaincode[resolvedPath] {
			continue
		}
		seenChaincode[resolvedPath] = true

		info, err := os.Stat(resolvedPath)
		if err != nil {
			return fmt.Errorf("chaincode '%s': %v", cc.Name, err)
		}

		if info.IsDir() {
			err = filepath.WalkDir(resolvedPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				rel, err := filepath.Rel(resolvedPath, path)
				if err != nil {
					return err
				}

				archivePath := filepath.ToSlash(filepath.Join(archiveBase, rel))
				if added[archivePath] {
					return nil
				}
				if err := addFileToZip(w, path, archivePath, password); err != nil {
					return err
				}
				added[archivePath] = true
				return nil
			})
			if err != nil {
				return fmt.Errorf("chaincode '%s': %v", cc.Name, err)
			}
		} else {
			archivePath := filepath.ToSlash(archiveBase)
			if added[archivePath] {
				continue
			}
			if err := addFileToZip(w, resolvedPath, archivePath, password); err != nil {
				return fmt.Errorf("chaincode '%s': %v", cc.Name, err)
			}
			added[archivePath] = true
		}
	}

	fmt.Println("✅ Đã nén xong thành công!")
	return nil
}

func resolveChaincodePath(pkgPath string, rootDir string) (string, string, error) {
	if pkgPath == "" {
		return "", "", fmt.Errorf("đường dẫn smart contract trống")
	}

	cleaned := filepath.Clean(pkgPath)
	modulePrefix := "khoai-chain/"
	archiveBase := filepath.ToSlash(cleaned)
	pathForAbs := cleaned

	if strings.HasPrefix(cleaned, modulePrefix) {
		trimmed := strings.TrimPrefix(cleaned, modulePrefix)
		archiveBase = filepath.ToSlash(trimmed)
		pathForAbs = trimmed
	}

	if filepath.IsAbs(cleaned) {
		pathForAbs = cleaned
		archiveBase = filepath.ToSlash(filepath.Base(cleaned))
	} else if rootDir != "" {
		pathForAbs = filepath.Join(rootDir, pathForAbs)
	}

	absPath, err := filepath.Abs(pathForAbs)
	if err != nil {
		return "", "", err
	}

	if archiveBase == "" {
		archiveBase = filepath.ToSlash(filepath.Base(absPath))
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", "", fmt.Errorf("không tìm thấy smart contract tại %s", absPath)
	}

	return absPath, archiveBase, nil
}

func addFileToZip(w *zip.Writer, filePath, archivePath, password string) error {
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := w.Encrypt(archivePath, password, zip.AES256Encryption)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)
	return err
}

func addBytesToZip(w *zip.Writer, archivePath string, data []byte, password string) error {
	writer, err := w.Encrypt(archivePath, password, zip.AES256Encryption)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}
