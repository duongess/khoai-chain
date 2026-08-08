package config

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var sourceCode embed.FS

func SetSourceCode(source embed.FS) {
	sourceCode = source
}

// LoadBuilderConfig loads the configuration from khoai.yaml.
// If the file does not exist, it returns a default configuration.
// It also applies defaults for any missing optional sections.
func LoadBuilderConfig(configPath string) (*BuilderConfig, error) {
	var cfg BuilderConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("khoai.yaml not found, using default configuration.")
			cfg = *GetDefaultBuilderConfig()
		} else {
			return nil, fmt.Errorf("could not read config file %s: %w", configPath, err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("error parsing YAML file %s: %w", configPath, err)
		}
	}

	// Apply defaults for missing sections
	ApplyDefaults(&cfg)

	// Validate the final configuration
	if err := ValidateBuilderConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// --- New Configuration Models ---

// BuilderConfig is the root configuration structure from khoai.yaml
type BuilderConfig struct {
	Network       *NetworkConfig       `yaml:"network,omitempty"`
	Docker        *DockerConfig        `yaml:"docker,omitempty"`
	Organizations []OrganizationConfig `yaml:"organizations"`
}

// NetworkConfig defines the overall network properties.
type NetworkConfig struct {
	Name   string `yaml:"name,omitempty"`
	Domain string `yaml:"domain,omitempty"`
}

// DockerConfig holds all settings for building Docker images.
type DockerConfig struct {
	ImageBase string `yaml:"image_base,omitempty"`
	ImageTag  string `yaml:"image_tag,omitempty"`
	Registry  string `yaml:"registry,omitempty"`
}

// OrganizationConfig represents a single organization in the network.
type OrganizationConfig struct {
	DisplayName string              `yaml:"display_name"`
	Metadata    *MetadataConfig     `yaml:"metadata,omitempty"`
	Chaincodes  []ChaincodeConfig   `yaml:"chaincodes"`
	Nodes       []RuntimeNodeConfig `yaml:"nodes"`
}

// MetadataConfig contains descriptive information about an organization.
type MetadataConfig struct {
	Description string `yaml:"description,omitempty"`
	Website     string `yaml:"website,omitempty"`
}

// RuntimeNodeConfig defines a single blockchain node server.
type RuntimeNodeConfig struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
	Endpoint    string `yaml:"endpoint"`
}

// ChaincodeConfig defines a smart contract.
type ChaincodeConfig struct {
	Name    string `yaml:"name,omitempty"`
	Package string `yaml:"package,omitempty"`
}

// MainTemplateData is used for generating main.go.
type MainTemplateData struct {
	Imports       []string
	Registrations []string
}

// GetDefaultBuilderConfig returns a default configuration for the network.
func GetDefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		Network: &NetworkConfig{
			Name:   "Khoai_Network",
			Domain: "khoai.local",
		},
		Docker: &DockerConfig{
			ImageBase: "golang:1.22-alpine",
			ImageTag:  "latest",
			Registry:  "registry.duongess.com/khoai-chain",
		},
		Organizations: []OrganizationConfig{
			{
				DisplayName: "DefaultOrg",
				Nodes: []RuntimeNodeConfig{
					{
						ID:          "node1",
						DisplayName: "Default Node 1",
						Endpoint:    "0.0.0.0:9000",
					},
				},
			},
		},
	}
}

// ApplyDefaults fills in missing configuration sections with default values.
func ApplyDefaults(cfg *BuilderConfig) {
	defaultCfg := GetDefaultBuilderConfig()
	if cfg.Network == nil {
		cfg.Network = defaultCfg.Network
	}
	if cfg.Docker == nil {
		cfg.Docker = defaultCfg.Docker
	}
	if len(cfg.Organizations) == 0 {
		cfg.Organizations = defaultCfg.Organizations
	}
}

// ValidateBuilderConfig checks the builder configuration for correctness.
func ValidateBuilderConfig(cfg *BuilderConfig) error {
	orgNames := make(map[string]bool)
	for _, org := range cfg.Organizations {
		if org.DisplayName == "" {
			return fmt.Errorf("organization display_name cannot be empty")
		}
		if orgNames[org.DisplayName] {
			return fmt.Errorf("duplicate organization display_name: %s", org.DisplayName)
		}
		orgNames[org.DisplayName] = true

		nodeIDs := make(map[string]bool)
		for _, node := range org.Nodes {
			if node.ID == "" {
				return fmt.Errorf("node id cannot be empty in organization %s", org.DisplayName)
			}
			if node.Endpoint == "" {
				return fmt.Errorf("node endpoint cannot be empty for node %s in organization %s", node.ID, org.DisplayName)
			}
			if _, _, err := net.SplitHostPort(node.Endpoint); err != nil {
				return fmt.Errorf("invalid endpoint format for node %s in organization %s: %s", node.ID, org.DisplayName, node.Endpoint)
			}
			if nodeIDs[node.ID] {
				return fmt.Errorf("duplicate node id '%s' in organization %s", node.ID, org.DisplayName)
			}
			nodeIDs[node.ID] = true
		}
	}
	return nil
}

// --- LOGIC TO GENERATE CONFIG FILE FOR EACH NODE ---

// GenerateNodeArtifacts creates the config.yaml, Dockerfile, and main.go for a single node.
func GenerateNodeArtifacts(nodeDir string, node RuntimeNodeConfig, org OrganizationConfig, cfg *BuilderConfig) error {
	// A. Generate config.yaml for this node (to be included in the Image)
	// Note: In Docker, the host is usually bound to 0.0.0.0

	_, port, err := net.SplitHostPort(node.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint format for node %s: %s", node.ID, node.Endpoint)
	}
	listenEndpoint := fmt.Sprintf("0.0.0.0:%s", port)

	uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(org.DisplayName), node.ID)

	// This struct defines the content of the generated runtime config.yaml.
	// It includes new fields for organization/node info while retaining
	// old fields for compatibility with the existing runtime.
	type RuntimeConfigContent struct {
		NodeName     string            `yaml:"node_name"`
		DBPath       string            `yaml:"db_path"`
		Chaincodes   []ChaincodeConfig `yaml:"chaincodes"`
		Organization string            `yaml:"organization"`
		NodeID       string            `yaml:"node_id"`
		DisplayName  string            `yaml:"display_name"`
		Endpoint     string            `yaml:"endpoint"`
	}

	finalConfig := RuntimeConfigContent{
		NodeName:   uniqueNodeName,
		DBPath:     "/app/data",
		Chaincodes: org.Chaincodes, // Contracts are inherited from the organization
	}
	finalConfig.Organization = org.DisplayName
	finalConfig.NodeID = node.ID
	finalConfig.DisplayName = node.DisplayName
	finalConfig.Endpoint = listenEndpoint

	configContent, err := yaml.Marshal(finalConfig)
	if err != nil {
		return fmt.Errorf("error marshalling config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// Generate the main.go file specific to this node's chaincodes
	if err := generateMainFile(nodeDir, org.Chaincodes); err != nil {
		return fmt.Errorf("error generating main.go: %v", err)
	}

	// B. Generate Dockerfile (Multi-stage build)
	dockerfileTmpl := `
# --- Stage 1: Builder ---
FROM {{.ImageBase}} AS builder
WORKDIR /app

# 1. Copy go.mod and go.sum to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy the rest of the source code
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY examples ./examples

# 3. Copy the generated main.go, overwriting the placeholder from the previous step.
#    This ensures the correct main file (with proper contract imports and config path) is used.
COPY build/{{.NodeName}}/main.go ./cmd/node/main.go

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
		"NodeName":  uniqueNodeName,
		"Port":      port,
		"ImageBase": cfg.Docker.ImageBase,
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

// GenerateDockerCompose creates the docker-compose.yaml file for the entire network.
func GenerateDockerCompose(baseDir string, cfg *BuilderConfig) error {
	// We need to create a flat list of nodes for the template.
	type ComposeNodeInfo struct {
		Name string // unique name: vingroup-hn
		Port string
	}
	var allNodes []ComposeNodeInfo

	for _, org := range cfg.Organizations {
		for _, node := range org.Nodes {
			_, port, err := net.SplitHostPort(node.Endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint for node %s-%s: %v", org.DisplayName, node.ID, err)
			}
			allNodes = append(allNodes, ComposeNodeInfo{
				Name: fmt.Sprintf("%s-%s", sanitize(org.DisplayName), node.ID),
				Port: port,
			})
		}
	}

	// The template data needs to be a struct that contains all the info.
	type ComposeTemplateData struct {
		NetworkName string
		Registry    string
		ImageTag    string
		Nodes       []ComposeNodeInfo
	}

	templateData := ComposeTemplateData{
		NetworkName: cfg.Network.Name,
		Registry:    cfg.Docker.Registry,
		ImageTag:    cfg.Docker.ImageTag,
		Nodes:       allNodes,
	}

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
	return t.Execute(f, templateData)
}

// Generate main file
func generateMainFile(outputDir string, chaincodes []ChaincodeConfig) error {

	mainTmpl := `package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	// Import core
	"khoai-chain/examples"
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/p2p"
	"khoai-chain/pkg/cli"

	{{range .Imports}}	"{{.}}"
	{{end}}
)

var (
	BuiltInNodeName string = "Unknown Node"
)

func main() {
	// 1. Setup config file discovery (logic to find it next to the exe)
	defaultConfigPath := filepath.Join("/app", "config.yaml")

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
		srv := p2p.NewServer(conf.Endpoint, contractManager)
		go srv.Start()

		// 6. Block main thread to keep the server running forever
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
		regLine := fmt.Sprintf("%s.%s()", pkgName)
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

// sanitize creates a filesystem-friendly name.
func sanitize(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}
