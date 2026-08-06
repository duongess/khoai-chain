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
FROM {{.ImageBase}} AS builder
WORKDIR /app

# 1. Copy generated main.go first. A change here MUST trigger a rebuild.
COPY build/{{.NodeName}}/main.go ./cmd/node/main.go

# 2. Copy go.mod and go.sum to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# 3. Copy the rest of the source code
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY examples ./examples

# 4. Build the application binary
RUN go build -o /khoai-node ./cmd/node

# --- Stage 2: Runner (Run the application) ---
FROM alpine:3.19
WORKDIR /app

# 5. Install necessary system libraries (if any)
RUN apk add --no-cache ca-certificates libc6-compat

# 6. Copy the final binary from the builder stage and the config file
COPY --from=builder /khoai-node .
COPY build/{{.NodeName}}/config.yaml .

# 7. Setup for running
RUN mkdir -p /app/data
EXPOSE {{.Port}}

CMD ["./khoai-node", "run"]
`
	// Template data
	data := map[string]interface{}{
		"NodeName":  node.Name,
		"Port":      node.Port,
		"ImageBase": net.ImageBase,
	}

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
	"os"
	"path/filepath"
	"time"

	// Import core
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/p2p"
	"khoai-chain/pkg/cli"
	"khoai-chain/examples"

	{{range .Imports}}	"{{.}}"
	{{end}}
)

var (
	BuiltInNodeName string = "Unknown Node"
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

	// --- COMMAND: RUN (The only command for the node) ---
	nodeCLI.AddCommand("run", "Run the blockchain node", func(args []string) error {
		// 1. Load Config
		conf, err := config.LoadConfig(*configPathFlag)
		if err != nil {
			fmt.Printf("Could not read config file at: %s\n", *configPathFlag)
			absPath, _ := filepath.Abs(*configPathFlag)
			fmt.Printf("   (Absolute path: %s)\n", absPath)
			os.Exit(1)
		}

		fmt.Println("========================================")
		fmt.Printf("KHOAI CHAIN NODE: %s\n", conf.NodeName)
		fmt.Printf("Config File: %s\n", *configPathFlag)
		fmt.Printf("Database Path: %s\n", conf.DBPath)
		fmt.Println("========================================")

		// 2. Initialize DB
		db := database.InitDB(conf.DBPath)
		defer db.Close()

		// 3. Initialize Blockchain
		chain := core.InitBlockchain(db)

		// 4. Initialize Smart Contract Manager
		contractManager := contract.NewManager(chain)

		// Register contracts
		contractManager.RegisterApp(examples.NewUsageExamples())
		{{range .Registrations}}
		contractManager.RegisterApp({{.}})
		{{end}}

		fmt.Printf("- Blockchain Height: %d\n", chain.GetBestHeight())

		// 5. Initialize P2P Server
		srv := p2p.NewServer(conf.Port, contractManager)
		go srv.Start()

		// 6. Connect to Peers (After a short delay)
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

		// 7. Block main thread to keep the server running forever
		select {}
	})

	nodeCLI.Run()
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
