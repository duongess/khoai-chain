package config

import (
	"fmt"
	"html/template"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadBuilderConfig loads the configuration from khoai-config.yaml.
// If the file does not exist, it returns a default configuration.
// It also applies defaults for any missing optional sections.
func LoadBuilderConfig(configPath string) (*BuilderConfig, error) {
	var cfg BuilderConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("khoai-config.yaml not found, using default configuration.")
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

// BuilderConfig is the root configuration structure from khoai-config.yaml
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

// GenerateOrganizationArtifacts creates all necessary files for a single organization.
func GenerateOrganizationArtifacts(orgDir string, org OrganizationConfig, cfg *BuilderConfig) error {
	// 1. Create the base directory for the organization
	if err := os.MkdirAll(orgDir, 0755); err != nil {
		return fmt.Errorf("could not create organization directory %s: %w", orgDir, err)
	}

	// 2. Generate organization.yaml (contains only this org's config)
	orgYAML, err := yaml.Marshal(org)
	if err != nil {
		return fmt.Errorf("error marshalling organization config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(orgDir, "organization.yaml"), orgYAML, 0644); err != nil {
		return fmt.Errorf("error writing organization.yaml: %w", err)
	}

	// 3. Generate .version file
	// The version is determined during `khoai generate` and should be passed down.
	// For now, we assume a .version file exists in the root build dir.
	versionData, err := os.ReadFile(filepath.Join(BuildDir, ".version"))
	if err == nil {
		if err := os.WriteFile(filepath.Join(orgDir, ".version"), versionData, 0644); err != nil {
			return fmt.Errorf("error writing .version file for org: %w", err)
		}
	} // If it fails, we proceed without it, package command will fail later which is fine.

	// 4. Generate khoai-config.yaml (contains network and docker info)
	rootCfg := BuilderConfig{
		Network: cfg.Network,
		Docker:  cfg.Docker,
	}
	rootYAML, err := yaml.Marshal(rootCfg)
	if err != nil {
		return fmt.Errorf("error marshalling root config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(orgDir, ConfigFileName), rootYAML, 0644); err != nil {
		return fmt.Errorf("error writing khoai-config.yaml: %w", err)
	}

	// 5. Create empty contracts directory
	if err := os.MkdirAll(filepath.Join(orgDir, "contracts"), 0755); err != nil {
		return fmt.Errorf("could not create contracts directory: %w", err)
	}

	// 6. Generate artifacts for each node within the organization
	nodesBaseDir := filepath.Join(orgDir, "nodes")
	if err := os.MkdirAll(nodesBaseDir, 0755); err != nil {
		return fmt.Errorf("could not create nodes directory: %w", err)
	}

	for _, node := range org.Nodes {
		nodeDir := filepath.Join(nodesBaseDir, node.ID)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return err
		}
		// Pass the unique name for the Dockerfile template
		uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(org.DisplayName), node.ID)
		if err := GenerateNodeArtifacts(nodeDir, node, org, cfg, uniqueNodeName); err != nil {
			return fmt.Errorf("error creating files for node %s: %w", node.ID, err)
		}
	}

	return nil
}

// GenerateNodeArtifacts creates the config.yaml, Dockerfile, and main.go for a single node.
func GenerateNodeArtifacts(nodeDir string, node RuntimeNodeConfig, org OrganizationConfig, cfg *BuilderConfig, uniqueNodeName string) error {
	sourceArchive, err := sourceArchiveFromVersionFile(filepath.Join(BuildDir, ".version"))
	if err != nil {
		return err
	}

	// A. Generate config.yaml for this node (to be included in the Image)
	// Note: In Docker, the host is usually bound to 0.0.0.0

	_, p2pPort, err := net.SplitHostPort(node.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint format for node %s: %s", node.ID, node.Endpoint)
	}
	httpPort := "9000"

	// This struct defines the content of the generated runtime config.yaml.
	// It includes new fields for organization/node info while retaining
	// old fields for compatibility with the existing runtime.
	type RuntimeConfigContent struct {
		NodeName           string            `yaml:"node_name"`
		DBPath             string            `yaml:"db_path"`
		Chaincodes         []ChaincodeConfig `yaml:"chaincodes"`
		Organization       string            `yaml:"organization"`
		NodeID             string            `yaml:"node_id"`
		DisplayName        string            `yaml:"display_name"`
		HTTPListenEndpoint string            `yaml:"http_listen"`
		HTTPEndpoint       string            `yaml:"http_endpoint"`
		P2PListenEndpoint  string            `yaml:"p2p_listen"`
		P2PEndpoint        string            `yaml:"p2p_endpoint"`
	}

	finalConfig := RuntimeConfigContent{
		NodeName:   uniqueNodeName,
		DBPath:     "/app/data",
		Chaincodes: org.Chaincodes, // Contracts are inherited from the organization
	}
	finalConfig.Organization = org.DisplayName
	finalConfig.NodeID = node.ID
	finalConfig.DisplayName = node.DisplayName
	finalConfig.HTTPListenEndpoint = fmt.Sprintf("0.0.0.0:%s", httpPort)
	finalConfig.HTTPEndpoint = fmt.Sprintf("%s:%s", uniqueNodeName, httpPort)
	finalConfig.P2PListenEndpoint = fmt.Sprintf("0.0.0.0:%s", p2pPort)
	finalConfig.P2PEndpoint = fmt.Sprintf("%s:%s", uniqueNodeName, p2pPort)

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

	// B. Generate Dockerfile (Single-stage for 'go run')
	dockerfileTmpl := `
# --- Single-Stage Build for Development/go run ---
FROM {{.ImageBase}}
WORKDIR /app

# Cai dat unzip de xu ly file nguon
RUN apk add --no-cache unzip

# Build context hien tai la thu muc "build/". Goi truc tiep den dist/
COPY dist/{{.SourceArchive}} ./src.zip

# Giai nen code vao /app va xoa zip de toi uu layer
RUN unzip src.zip -d . && rm src.zip

# Tai thu vien phu thuoc cho du an
RUN go mod download

# Goi truc tiep den organizations/ tu build context
COPY organizations/{{.OrgName}}/nodes/{{.NodeID}}/main.go ./cmd/node/main.go
COPY organizations/{{.OrgName}}/nodes/{{.NodeID}}/config.yaml ./config.yaml
COPY organizations/{{.OrgName}}/contracts/ ./contracts/

# Tao thu muc luu tru va mo port
RUN mkdir -p /app/data
EXPOSE {{.HTTPPort}} {{.P2PPort}}

# Khoi chay node
CMD ["go", "run", "./cmd/node/main.go", "--config", "/app/node-config/config.yaml"]
`
	// Template data
	data := map[string]interface{}{
		"HTTPPort":      httpPort,
		"P2PPort":       p2pPort,
		"ImageBase":     cfg.Docker.ImageBase,
		"OrgName":       sanitize(org.DisplayName),
		"NodeID":        node.ID,
		"SourceArchive": sourceArchive,
	}

	t, _ := template.New("dockerfile").Parse(dockerfileTmpl)
	f, err := os.Create(filepath.Join(nodeDir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

// GenerateWorkspaceNodeArtifacts creates artifacts for a node within a workspace context.
// The key difference is the Dockerfile paths, which are relative to the workspace root.
func GenerateWorkspaceNodeArtifacts(nodeDir string, node RuntimeNodeConfig, org OrganizationConfig, cfg *BuilderConfig, uniqueNodeName string) error {
	workspaceDir := filepath.Dir(filepath.Dir(nodeDir))
	sourceArchive, err := sourceArchiveFromVersionFile(filepath.Join(workspaceDir, ".version"))
	if err != nil {
		return err
	}

	// A. Generate config.yaml for this node (to be included in the Image)
	_, p2pPort, err := net.SplitHostPort(node.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint format for node %s: %s", node.ID, node.Endpoint)
	}
	httpPort := "9000"

	type RuntimeConfigContent struct {
		NodeName           string            `yaml:"node_name"`
		DBPath             string            `yaml:"db_path"`
		Chaincodes         []ChaincodeConfig `yaml:"chaincodes"`
		Organization       string            `yaml:"organization"`
		NodeID             string            `yaml:"node_id"`
		DisplayName        string            `yaml:"display_name"`
		HTTPListenEndpoint string            `yaml:"http_listen"`
		HTTPEndpoint       string            `yaml:"http_endpoint"`
		P2PListenEndpoint  string            `yaml:"p2p_listen"`
		P2PEndpoint        string            `yaml:"p2p_endpoint"`
	}

	finalConfig := RuntimeConfigContent{
		NodeName:           uniqueNodeName,
		DBPath:             "/app/data",
		Chaincodes:         org.Chaincodes,
		Organization:       org.DisplayName,
		NodeID:             node.ID,
		DisplayName:        node.DisplayName,
		HTTPListenEndpoint: fmt.Sprintf("0.0.0.0:%s", httpPort),
		HTTPEndpoint:       fmt.Sprintf("%s:%s", uniqueNodeName, httpPort),
		P2PListenEndpoint:  fmt.Sprintf("0.0.0.0:%s", p2pPort),
		P2PEndpoint:        fmt.Sprintf("%s:%s", uniqueNodeName, p2pPort),
	}

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

	// B. Generate Dockerfile (Multi-stage build for WORKSPACE)
	dockerfileTmpl := `
# --- Single-Stage Build for Development/go run ---
FROM {{.ImageBase}}
WORKDIR /app

# Cai dat unzip de xu ly file nguon
RUN apk add --no-cache unzip

# Build context hien tai la thu muc workspace. Goi truc tiep den dist/
COPY dist/{{.SourceArchive}} ./src.zip

# Giai nen code vao /app va xoa zip de toi uu layer
RUN unzip src.zip -d . && rm src.zip

# Tai thu vien phu thuoc cho du an
RUN go mod download

# Goi truc tiep den node cua organization workspace hien tai
COPY nodes/{{.NodeID}}/main.go ./cmd/node/main.go
COPY nodes/{{.NodeID}}/config.yaml ./config.yaml
COPY contracts/ ./contracts/

# Tao thu muc luu tru va mo port
RUN mkdir -p /app/data
EXPOSE {{.HTTPPort}} {{.P2PPort}}

# Khoi chay node
CMD ["go", "run", "./cmd/node/main.go", "--config", "/app/node-config/config.yaml"]
`
	// Template data
	data := map[string]interface{}{
		"NodeID":        node.ID,
		"HTTPPort":      httpPort,
		"P2PPort":       p2pPort,
		"ImageBase":     cfg.Docker.ImageBase,
		"SourceArchive": sourceArchive,
	}

	t, _ := template.New("dockerfile-workspace").Parse(dockerfileTmpl)
	f, err := os.Create(filepath.Join(nodeDir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

// sourceArchiveFromVersionFile returns the source archive name produced by
// install.sh for the version pinned in a .version file.
func sourceArchiveFromVersionFile(versionFile string) (string, error) {
	versionData, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("could not read source version from %q: %w", versionFile, err)
	}

	version := strings.TrimSpace(string(versionData))
	if version == "" || filepath.Base(version) != version {
		return "", fmt.Errorf("invalid source version in %q", versionFile)
	}

	return fmt.Sprintf("khoai-src-%s.zip", version), nil
}

// --- LOGIC TO GENERATE DOCKER-COMPOSE ---

// GenerateDockerCompose creates the docker-compose.yaml file for the entire network.
func GenerateDockerCompose(baseDir string, cfg *BuilderConfig) error {
	// We need to create a flat list of nodes for the template.
	type ComposeNodeInfo struct {
		Name     string // unique name: vingroup-hn
		OrgName  string // sanitized org name: vingroup
		NodeID   string // node id: hn
		P2PPort  string // e.g., 8080
		HTTPPort string // e.g., 18080
	}
	var allNodes []ComposeNodeInfo

	for _, org := range cfg.Organizations {
		sanitizedOrgName := sanitize(org.DisplayName)
		for _, node := range org.Nodes {
			_, p2pPort, err := net.SplitHostPort(node.Endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint for node %s-%s: %v", org.DisplayName, node.ID, err)
			}

			allNodes = append(allNodes, ComposeNodeInfo{
				Name:     fmt.Sprintf("%s-%s", sanitizedOrgName, node.ID),
				OrgName:  sanitizedOrgName,
				NodeID:   node.ID,
				P2PPort:  p2pPort,
				HTTPPort: "9000",
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
      context: .
      dockerfile: ./organizations/{{.OrgName}}/nodes/{{.NodeID}}/Dockerfile
    image: {{$.Registry}}/{{.Name}}:{{$.ImageTag}}
    container_name: {{.Name}}
    ports:
      - "{{.P2PPort}}:{{.P2PPort}}"
      - "{{.HTTPPort}}:9000"
    expose:
      - "{{.P2PPort}}"
      - "9000"
    volumes:
      - ./data/{{.Name}}:/app/data
      - ./organizations/{{.OrgName}}/nodes/{{.NodeID}}:/app/node-config
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

// GenerateWorkspaceDockerCompose creates a docker-compose.yaml file for a single organization workspace.
func GenerateWorkspaceDockerCompose(baseDir string, cfg *BuilderConfig) error {
	type ComposeNodeInfo struct {
		Name     string // unique name: vingroup-hn
		NodeID   string // node id: hn
		P2PPort  string // e.g., 8080
		HTTPPort string // e.g., 18080
	}
	var allNodes []ComposeNodeInfo

	// In a workspace, there's only one organization
	org := cfg.Organizations[0]
	sanitizedOrgName := sanitize(org.DisplayName)
	for _, node := range org.Nodes {
		_, p2pPort, err := net.SplitHostPort(node.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint for node %s-%s: %v", org.DisplayName, node.ID, err)
		}

		allNodes = append(allNodes, ComposeNodeInfo{
			Name:     fmt.Sprintf("%s-%s", sanitizedOrgName, node.ID),
			NodeID:   node.ID,
			P2PPort:  p2pPort,
			HTTPPort: "9000",
		})
	}

	type ComposeTemplateData struct {
		NetworkName string
		Registry    string
		ImageTag    string
		Nodes       []ComposeNodeInfo
	}

	templateData := ComposeTemplateData{
		NetworkName: "khoai-network",
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
      context: .
      dockerfile: ./nodes/{{.NodeID}}/Dockerfile
    image: {{$.Registry}}/{{.Name}}:{{$.ImageTag}}
    container_name: {{.Name}}
    ports:
      - "{{.P2PPort}}:{{.P2PPort}}"
      - "{{.HTTPPort}}:9000"
    expose:
      - "{{.P2PPort}}"
      - "9000"
    volumes:
      - ./data/{{.Name}}:/app/data
      - ./nodes/{{.NodeID}}:/app/node-config
    networks:
      - {{$.NetworkName}}
    restart: always
{{end}}
`
	t, err := template.New("compose-workspace").Parse(composeTmpl)
	if err != nil {
		return err
	}
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
	"net/http"
	"os"
	"path/filepath"

	// Import core
	"khoai-chain/examples"
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/p2p"

	{{range .Imports}}	"{{.}}"
	{{end}}
)

var (
	BuiltInNodeName string = "Unknown Node"
)

func main() {
	defaultConfigPath := filepath.Join("/app", "config.yaml")

	// Parse flags to get the config path
	configPathFlag := flag.String("config", defaultConfigPath, "Path to the configuration file")
	flag.Parse()

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
	// The P2P server handles the core blockchain protocol (block and transaction gossip).
	srv := p2p.NewServer(conf.P2PEndpoint, contractManager)
	srv.ConfigurePersistence(conf, *configPathFlag)
	go srv.Start()

	// 6. Initialize HTTP Control Plane
	// The HTTP server exposes API endpoints for node management (/join, /peers, etc.).
	// We assume the 'p2p.Server' struct also implements http.Handler to serve these requests.
	fmt.Printf("HTTP API server listening on %s\n", conf.HTTPListenEndpoint)
	go func() {
		_ = http.ListenAndServe(conf.HTTPListenEndpoint, srv)
	}()

	// 7. Block main thread to keep the server running forever
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

		// Convert a name like "sample-contract" to a Go constructor function name like "NewSampleContract"
		// 1. Split by "-": ["sample", "contract"]
		// 2. Capitalize each part: ["Sample", "Contract"]
		// 3. Join and prepend "New": "NewSampleContract"
		var parts = strings.Split(cc.Name, "-")
		var goNameParts []string
		for _, part := range parts {
			if len(part) > 0 {
				goNameParts = append(goNameParts, strings.ToUpper(string(part[0]))+part[1:])
			}
		}
		funcName := "New" + strings.Join(goNameParts, "")

		regLine := fmt.Sprintf("%s.%s()", pkgName, funcName)
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
