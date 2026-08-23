package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"khoai-chain/internal/config"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// generateArtifacts đọc cấu hình builder chính và tạo artifacts cho tất cả các tổ chức.
func generateArtifacts(configPath string) error {
	fmt.Println("Generating build artifacts...")
	builderConf, err := config.LoadBuilderConfig(configPath)
	if err != nil {
		return err
	}

	orgsBaseDir := filepath.Join(config.BuildDir, config.OrgsDir)
	if err := os.MkdirAll(orgsBaseDir, 0755); err != nil {
		return err
	}

	for _, org := range builderConf.Organizations {
		fmt.Printf("- Generating artifacts for organization: %s\n", org.DisplayName)
		orgDir := filepath.Join(orgsBaseDir, sanitize(org.DisplayName))
		if err := config.GenerateOrganizationArtifacts(orgDir, org, builderConf); err != nil {
			return fmt.Errorf("error generating artifacts for organization %s: %w", org.DisplayName, err)
		}
	}

	fmt.Println("- Generating main docker-compose.yaml")
	if err := config.GenerateDockerCompose(config.BuildDir, builderConf); err != nil {
		return fmt.Errorf("error creating docker-compose.yaml file: %w", err)
	}

	fmt.Println("Artifact generation complete.")
	return nil
}

func isWorkspaceOrganizations() (bool, error) {
	if _, err := os.Stat("khoai-config.yaml"); err == nil {
		builderConf, err := config.LoadBuilderConfig("khoai-config.yaml")
		if err != nil {
			return false, fmt.Errorf("error loading builder config: %w", err)
		}
		if len(builderConf.Organizations) == 0 {
			fmt.Println("No organizations defined in khoai-config.yaml. Please define at least one organization.")
			return false, nil
		}
		return true, nil
	} else if os.IsNotExist(err) {
		fmt.Println("No khoai-config.yaml found in the current directory. Please run this command in a valid workspace.")
		return false, nil
	} else {
		fmt.Println("Error checking for khoai-config.yaml:", err)
		return false, err
	}
}

// downloadViaScript xử lý việc tải và giải nén mã nguồn từ GitHub.
func downloadViaScript(version string, targetDir string) (string, error) {
	fmt.Printf("Starting process to download source code version: %s\n", version)

	scriptURL := "https://raw.githubusercontent.com/duongess/khoai-chain/main/install.sh"
	shellCmd := fmt.Sprintf("curl -fsSL %s | bash -s -- %s %s", scriptURL, version, targetDir)
	cmd := exec.Command("bash", "-c", shellCmd)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error executing install.sh script: %w", err)
	}

	downloadedVersion := strings.TrimSpace(out.String())
	if downloadedVersion == "" {
		return "", fmt.Errorf("install.sh script did not output a version string")
	}
	return downloadedVersion, nil
}

// runCommand là một hàm trợ giúp để thực thi các lệnh shell và stream output.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Executing: %s\n", cmd.String())
	return cmd.Run()
}

// sanitize tạo ra một tên thân thiện với hệ thống file.
func sanitize(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}

func peerAPIAddress(address string) string { return address }
func peerAPI(address, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, "http://"+peerAPIAddress(address)+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("peer API: %s: %s", res.Status, strings.TrimSpace(string(data)))
	}

	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

// p2pToHTTP converts a P2P endpoint (e.g., "localhost:8080") to its corresponding
// control plane HTTP endpoint (e.g., "localhost:9000").
// In our single-host Docker simulation, we map the internal container port 9000
// to a host port derived from the P2P port (e.g., 8080 -> 18080).
func p2pToHTTP(p2pEndpoint string) string {
	p2pEndpoint = strings.TrimSpace(p2pEndpoint)

	// Case 1: endpoint chỉ có host, không có port.
	// Example: "node.example.com" -> "node.example.com:9000"
	if !strings.Contains(p2pEndpoint, ":") {
		return net.JoinHostPort(p2pEndpoint, "9000")
	}

	host, p2pPortStr, err := net.SplitHostPort(p2pEndpoint)
	if err != nil {
		// Fallback for endpoints like ":8080"
		if strings.HasPrefix(p2pEndpoint, ":") {
			host = "localhost"
			p2pPortStr = strings.TrimPrefix(p2pEndpoint, ":")
		} else {
			// Invalid endpoint; fallback to localhost:9000
			return "localhost:9000"
		}
	}

	// Case 2: no host or bind address is 0.0.0.0.
	// This is the local Docker/single-host simulation case.
	// Keep the existing convention: HTTP port = P2P port + 10000.
	if host == "" || host == "0.0.0.0" || host == "localhost" {
		p2pPort, err := strconv.Atoi(p2pPortStr)
		if err != nil {
			return "localhost:9000"
		}

		httpPort := p2pPort + 10000
		return net.JoinHostPort("localhost", strconv.Itoa(httpPort))
	}

	// Case 3: real hostname/IP.
	// P2P port and HTTP port are independent.
	// HTTP Control Plane always uses port 9000.
	return net.JoinHostPort(host, "9000")
}

// findNodeInternalAddressByEndpoint finds a node by its host-facing endpoint (e.g., "localhost:8082")
// and returns its internal Docker network address (e.g., "coteccons-hcm:8082").
func findNodeInternalAddressByEndpoint(configPath, hostEndpoint string) (string, error) {
	builderConf, err := config.LoadBuilderConfig(configPath)
	if err != nil {
		// This can happen in a workspace context where the main config is not in the root.
		// We can try to load from the current directory.
		if os.IsNotExist(err) {
			builderConf, err = config.LoadBuilderConfig("khoai-config.yaml")
		}
		if err != nil {
			return "", fmt.Errorf("could not load builder config from %s: %w", configPath, err)
		}
	}

	// Normalize endpoint to look for just the port
	_, port, err := net.SplitHostPort(hostEndpoint)
	if err != nil {
		if strings.HasPrefix(hostEndpoint, ":") {
			port = hostEndpoint[1:]
		} else {
			return "", fmt.Errorf("invalid endpoint format: %s", hostEndpoint)
		}
	}

	for _, org := range builderConf.Organizations {
		for _, node := range org.Nodes {
			_, nodePort, _ := net.SplitHostPort(node.Endpoint)
			if nodePort == port {
				// Found it. Construct the internal docker name.
				sanitizedOrgName := sanitize(org.DisplayName)
				internalName := fmt.Sprintf("%s-%s", sanitizedOrgName, node.ID)
				internalEndpoint := fmt.Sprintf("%s:%s", internalName, nodePort)
				return internalEndpoint, nil
			}
		}
	}

	return "", fmt.Errorf("no node found with endpoint port %s", port)
}

func generateWorkspaceNodeArtifacts(force bool) (int, error) {
	rootConf, err := config.LoadBuilderConfig(config.ConfigFileName)
	if err != nil {
		return 0, fmt.Errorf("could not load workspace khoai-config.yaml: %w", err)
	}
	orgData, err := os.ReadFile("organization.yaml")
	if err != nil {
		return 0, fmt.Errorf("could not load workspace organization.yaml: %w", err)
	}
	var orgConf config.OrganizationConfig
	if err := yaml.Unmarshal(orgData, &orgConf); err != nil {
		return 0, fmt.Errorf("could not parse workspace organization.yaml: %w", err)
	}

	nodesBaseDir := "nodes"
	nodesGenerated := 0
	for _, node := range orgConf.Nodes {
		nodeDir := filepath.Join(nodesBaseDir, node.ID)

		if _, err := os.Stat(nodeDir); err == nil && !force {
			fmt.Printf("- Node '%s' already exists, skipping.\n", node.ID)
			continue
		}

		fmt.Printf("- Generating artifacts for node: '%s'\n", node.ID)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return 0, fmt.Errorf("could not create directory for node %s: %w", node.ID, err)
		}

		uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(orgConf.DisplayName), node.ID)
		if err := config.GenerateWorkspaceNodeArtifacts(nodeDir, node, orgConf, rootConf, uniqueNodeName); err != nil {
			return 0, fmt.Errorf("error creating files for node %s: %w", node.ID, err)
		}
		nodesGenerated++
	}

	return nodesGenerated, nil
}

// createDefaultWorkspaceFiles tạo các file cấu hình mặc định cho một workspace.
func createDefaultWorkspaceFiles(dir string) error {
	defaultCfg := config.GetDefaultBuilderConfig()
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}
	defaultOrg := config.OrganizationConfig{
		DisplayName: filepath.Base(cwd),
		Nodes: []config.RuntimeNodeConfig{
			{ID: "node1", DisplayName: "Default Node", Endpoint: "localhost:9000"},
		},
	}

	orgYAML, err := yaml.Marshal(defaultOrg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "organization.yaml"), orgYAML, 0644); err != nil {
		return err
	}

	rootCfg := config.BuilderConfig{Network: defaultCfg.Network, Docker: defaultCfg.Docker}
	rootYAML, err := yaml.Marshal(rootCfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, config.ConfigFileName), rootYAML, 0644)
}

// generateWorkspaceCompose tạo file docker-compose.yaml trong một workspace.
func generateWorkspaceCompose(composePath string) error {
	rootConf, err := config.LoadBuilderConfig(config.ConfigFileName)
	if err != nil {
		return fmt.Errorf("could not load workspace khoai-config.yaml: %w", err)
	}

	orgData, err := os.ReadFile("organization.yaml")
	if err != nil {
		return fmt.Errorf("could not load workspace organization.yaml: %w", err)
	}
	var orgConf config.OrganizationConfig
	if err := yaml.Unmarshal(orgData, &orgConf); err != nil {
		return fmt.Errorf("could not parse workspace organization.yaml: %w", err)
	}

	workspaceBuilderConfig := &config.BuilderConfig{
		Network:       rootConf.Network,
		Docker:        rootConf.Docker,
		Organizations: []config.OrganizationConfig{orgConf},
	}

	if err := config.GenerateWorkspaceDockerCompose(".", workspaceBuilderConfig); err != nil {
		return err
	}
	fmt.Println("Generated workspace docker-compose.yaml")
	return nil
}
