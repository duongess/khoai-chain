package config

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

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

// --- LOGIC TO GENERATE CONFIG FILE FOR EACH NODE ---
func GenerateNodeArtifacts(nodeDir string, node NodeConfig, net NetworkConfig) error {
	// A. Generate config.yaml for this node (to be included in the Image)
	// Note: In Docker, the host is usually bound to 0.0.0.0

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
		return fmt.Errorf("error marshalling config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// Generate the main.go file specific to this node's chaincodes
	if err := generateMainFile(nodeDir, node.Chaincodes); err != nil {
		return fmt.Errorf("error generating main.go: %v", err)
	}

	// B. Generate Dockerfile (Multi-stage build)
	dockerfileTmpl := `
# --- Stage 1: Builder ---
FROM golang:1.22-alpine AS builder
WORKDIR /app

# 1. Copy go.mod and go.sum to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy the entire source code from the project root
COPY . .

# 3. Overwrite the generic main.go with the one generated for this specific node
# The path is relative to the build context (project root)
COPY build/{{.NodeName}}/main.go ./cmd/node/main.go

# 4. Build the application
RUN go build -o khoai-node ./cmd/node

# --- Stage 2: Runner (Run the application) ---
FROM alpine:3.19
WORKDIR /app

# 5. Install necessary system libraries
RUN apk add --no-cache ca-certificates libc6-compat

# 6. Copy only the 'khoai-node' binary and the config file
COPY --from=builder /app/khoai-node .
COPY build/{{.NodeName}}/config.yaml .

# 7. Setup for running
RUN mkdir -p /app/data
EXPOSE {{.Port}}

CMD ["./khoai-node", "run"]
`
	// Template data
	data := map[string]interface{}{"NodeName": node.Name, "Port": node.Port}

	t, _ := template.New("dockerfile").Parse(dockerfileTmpl)
	f, err := os.Create(filepath.Join(nodeDir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

// --- LOGIC TO GENERATE DOCKER-COMPOSE ---
func GenerateDockerCompose(baseDir string, net NetworkConfig) error {
	composeTmpl := `version: "3.9"

networks:
  {{.NetworkName}}:
    driver: bridge

services:
{{range .Nodes}}
  {{.Name}}:
    build:
      context: ..  # Build context is the project root, relative to this compose file
      dockerfile: ./build/{{.Name}}/Dockerfile # Path to the Dockerfile, relative to the context
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
	// 1. Setup config file discovery (logic to find it next to the exe)
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)
	defaultConfigPath := filepath.Join(exeDir, "config.yaml")

	// Parse flags to get the config path
	configPathFlag := flag.String("config", defaultConfigPath, "Path to the configuration file")
	flag.Parse()

	// 2. Initialize CLI
	nodeCLI := cli.NewCLI()

	// --- COMMAND: RUN (For Docker/Production) ---
	// This mode strictly respects the path in the config file (e.g., /app/data)
	nodeCLI.AddCommand("run", "Run node in Production (Docker) mode", func(args []string) error {
		fmt.Println("Mode: DOCKER / PRODUCTION")
		startNode(*configPathFlag, false)
		return nil
	})

	// --- COMMAND: DEV (For quick developer testing) ---
	// This mode forces the DB to be next to the exe, regardless of the config
	nodeCLI.AddCommand("dev", "Run node in Dev mode (DB saved next to the exe)", func(args []string) error {
		fmt.Println("Mode: DEVELOPMENT")
		startNode(*configPathFlag, true)
		return nil
	})

	nodeCLI.Run()
}

func startNode(configPath string, isDevMode bool) {
	// 1. Load Config
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Could not read config file at: %s\n", configPath)
		// Suggest absolute path for easier debugging
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("   (Absolute path: %s)\n", absPath)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("KHOAI CHAIN NODE: %s\n", BuiltInNodeName)
	fmt.Printf("Config File: %s\n", configPath)

	// 2. HANDLE DATABASE PATH (Most important logic here)
	finalDBPath := conf.DBPath

	if isDevMode {
		// LOGIC FOR DEV:
		dbName := filepath.Base(conf.DBPath)

		// And force it to be next to the exe file
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		finalDBPath = filepath.Join(exeDir, dbName)

		fmt.Println("Dev Override: Forcing DB to local directory")
	} else {
		// LOGIC FOR DOCKER / RUN:
		// Keep the config. If config is "/app/data", use it as is.
		fmt.Println("Docker Mode: Using DB path from config")
	}

	fmt.Printf("Database Path: %s\n", finalDBPath)
	fmt.Println("========================================")

	// 3. Initialize DB
	db := database.InitDB(finalDBPath)
	// Note: In server's infinite loop mode (select{}), this defer only runs on app shutdown (Ctrl+C)
	defer db.Close()

	// 4. Initialize Blockchain
	chain := core.InitBlockchain(db)

	// 5. Initialize Smart Contract Manager
	contractManager := contract.NewManager(chain)

	// Register example contracts (if needed)
	// contractManager.RegisterApp(examples.NewUsageExamples())
	// (Or use .Imports .Registrations from your template)
	contractManager.RegisterApp(examples.NewUsageExamples())

	fmt.Printf("- Blockchain Height: %d\n", chain.GetBestHeight())

	// 6. Initialize P2P Server
	srv := p2p.NewServer(conf.Port, contractManager)
	go srv.Start()

	// 7. Connect to Peers (After 2s)
	go func() {
		time.Sleep(2 * time.Second)
		if len(conf.Peers) > 0 {
			fmt.Println("Peers list in config:", conf.Peers)

			for _, peerAddr := range conf.Peers {
				fmt.Printf("- Connecting to peer: %s\n", peerAddr)
				srv.ConnectToPeer(peerAddr)
			}
		}
	}()

	// 8. Block main thread to keep the server running forever
	select {}
}
	`

	data := MainTemplateData{}
	seen := make(map[string]bool)

	for _, cc := range chaincodes {
		// Add import if not already present
		if !seen[cc.Package] {
			data.Imports = append(data.Imports, cc.Package)
			seen[cc.Package] = true
		}
		// Add registration line: bds.NewBDSContract()
		// Assume package name is the last part of the path (e.g., bds)
		pkgName := filepath.Base(cc.Package)
		regLine := fmt.Sprintf("%s.%s()", pkgName, cc.Constructor)
		data.Registrations = append(data.Registrations, regLine)
	}

	// Write file
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
