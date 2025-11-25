package config

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

type ConfigContent struct {
	NodeName   string            `yaml:"node_name"`
	Host       string            `yaml:"host"`
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

func LoadConfig(filePath string) (*ConfigContent, error) {
	// 1. Đọc nội dung file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy file config: %v", err)
	}

	// 2. Giải mã YAML vào struct
	var conf ConfigContent
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("lỗi định dạng YAML: %v", err)
	}

	return &conf, nil
}

// --- LOGIC SINH FILE CONFIG RIÊNG CHO TỪNG NODE ---
func GenerateNodeArtifacts(baseDir string, node NodeConfig, net NetworkConfig) error {
	nodeDir := filepath.Join(baseDir, node.Name)
	os.MkdirAll(nodeDir, 0755)

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

	err = generateMainFile(nodeDir, node.Chaincodes)
	if err != nil {
		return fmt.Errorf("lỗi sinh main.go: %v", err)
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
