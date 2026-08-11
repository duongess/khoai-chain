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

func peerAPIAddress(p2pAddress string) string {
	host, port, err := net.SplitHostPort(p2pAddress)
	if err != nil {
		return p2pAddress
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return p2pAddress
	}
	return net.JoinHostPort(host, strconv.Itoa(n+1))
}
func peerAPI(address, method, path string, body any) error {
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
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("peer API: %s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	fmt.Print(string(data))
	return nil
}
