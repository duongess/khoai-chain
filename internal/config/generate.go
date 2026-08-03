package config

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	utils "khoai-chain/internal/ulits"
	"os"
	"path/filepath"
	"strings"

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

	if err := generateMainFile(nodeDir, node.Chaincodes); err != nil {
		return fmt.Errorf("error generating main.go: %v", err)
	}

	if err := ZipFiles(nodeDir, node.Chaincodes); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(nodeDir, "main.go"))

	// B. Generate Dockerfile (Multi-stage build)
	dockerfileTmpl := `
# --- Stage 1: Builder (Go environment needed for khoai build to work) ---
FROM golang:1.22-alpine AS builder
WORKDIR /app

# 1. Install curl and bash to run the khoai installation command
RUN apk add --no-cache curl bash

# 2. Install Khoai CLI
RUN curl -fsSL https://raw.githubusercontent.com/duongess/khoaichain-sdk/main/install.sh | bash

# 3. Copy the protected zip file
COPY ./khoai_protected.zip .

# 4. Run the build command (This will succeed as Go is now available)
RUN khoai build .

# --- Stage 2: Runner (Run the application) ---
FROM alpine:3.19
WORKDIR /app

# 5. Install necessary system libraries
RUN apk add --no-cache ca-certificates libc6-compat

# 6. Copy only the 'khoai-node' binary built from Stage 1
COPY --from=builder /app/khoai-node .
COPY config.yaml .

# 7. Setup for running
RUN mkdir -p /app/data
EXPOSE 9002

CMD ["./khoai-node", "run"]
`
	// Template data
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
      context: ./{{.Name}}  # Points to the root folder containing the source code
      dockerfile: Dockerfile
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
		fmt.Println("🐳 Mode: DOCKER / PRODUCTION")
		startNode(*configPathFlag, false)
		return nil
	})

	// --- COMMAND: DEV (For quick testing by developers) ---
	nodeCLI.AddCommand("dev", "Run node in Dev mode (DB saved next to the exe)", func(args []string) error {
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
		fmt.Printf("❌ Could not read config file at: %s\n", configPath)
		// Suggest absolute path for easier debugging
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("   (Absolute path: %s)\n", absPath)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("🏭 KHOAI CHAIN NODE: %s\n", BuiltInNodeName)
	fmt.Printf("📂 Config File: %s\n", configPath)

	// 2. HANDLE DATABASE PATH (Most important logic here)
	finalDBPath := conf.DBPath

	if isDevMode {
		// LOGIC FOR DEV:
		dbName := filepath.Base(conf.DBPath)

		// And force it to be next to the exe file
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		finalDBPath = filepath.Join(exeDir, dbName)

		fmt.Println("🔧 Dev Override: Forcing DB to local directory")
	} else {
		// LOGIC FOR DOCKER / RUN:
		// Keep the config. If config is "/app/data", use it as is.
		fmt.Println("🐳 Docker Mode: Using DB path from config")
	}

	fmt.Printf("💾 Database Path: %s\n", finalDBPath)
	fmt.Println("========================================")

	// 3. Initialize DB
	db := database.InitDB(finalDBPath)
	// Note: In server's infinite loop mode (select{}), this defer only runs on app shutdown (Ctrl+C)
	defer db.Close()

	// 4. Initialize Blockchain
	chain := core.InitBlockchain(db)

	// 5. Initialize Smart Contract Manager
	contractManager := contract.NewManager(chain)

	fmt.Println("📦 Loading Chaincodes for this Node...")
	{{range .Registrations}}	contractManager.RegisterApp({{.}})
	{{end}}

	fmt.Printf("⛓️  Blockchain Height: %d\n", chain.GetBestHeight())

	// 6. Initialize P2P Server
	srv := p2p.NewServer(conf.Port, contractManager)
	go srv.Start()

	// 7. Connect to Peers (After 2s)
	go func() {
		time.Sleep(2 * time.Second)
		if len(conf.Peers) > 0 {
			fmt.Println("🌐 Peers list in config:", conf.Peers)

			for _, peerAddr := range conf.Peers {
				targetAddr := peerAddr

				if isDevMode {
					parts := strings.Split(peerAddr, ":")
					// Ensure correct host:port format
					if len(parts) == 2 {
						port := parts[1]

						// Create a new address pointing to localhost
						targetAddr = fmt.Sprintf("localhost:%s", port)
					}
				}

				// Connect to the address (processed or original)
				srv.ConnectToPeer(targetAddr)
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

// Zip main.go and smart contracts into khoai_protected.zip for each node
func ZipFiles(nodeDir string, chaincodes []ChaincodeConfig) error {

	password, err := utils.GetEnv("KHOAI_PASS")
	if err != nil {
		return fmt.Errorf("❌ Error: KHOAI_PASS environment variable not set")
	}

	if password == "" {
		return fmt.Errorf("❌ Error: KHOAI_PASS environment variable not set")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current directory: %v", err)
	}

	mainPath := filepath.Join(nodeDir, "main.go")
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("main.go not found at %s: %v", mainPath, err)
	}

	outputPath := filepath.Join(nodeDir, "khoai_protected.zip")
	fmt.Printf("🔒 Zipping main + smart contracts into '%s' with password...\n", outputPath)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create zip file: %v", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	added := make(map[string]bool)

	// 1) Zip all embedded source code (except main for replacement)
	if err := fs.WalkDir(sourceCode, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == "." {
			return nil
		}

		archivePath := filepath.ToSlash(strings.TrimPrefix(path, "./"))
		if archivePath == "cmd/node/main.go" {
			return nil // will be replaced by the generated main
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
		return fmt.Errorf("error during zipping of embedded source: %v", err)
	}

	// 2) Replace cmd/node/main.go with the generated main file for the node
	if err := addFileToZip(w, mainPath, "cmd/node/main.go", password); err != nil {
		return fmt.Errorf("error zipping %s: %v", mainPath, err)
	}
	added["cmd/node/main.go"] = true

	// 3) Zip additional smart contracts from config (customer-specific)
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

	fmt.Println("✅ Zipping completed successfully!")
	return nil
}

func resolveChaincodePath(pkgPath string, rootDir string) (string, string, error) {
	if pkgPath == "" {
		return "", "", fmt.Errorf("smart contract path is empty")
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
		return "", "", fmt.Errorf("smart contract not found at %s", absPath)
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
