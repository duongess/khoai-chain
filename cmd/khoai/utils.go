package main

import (
	"bufio"
	"bytes"
	"fmt"
	"khoai-chain/internal/config"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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

// sendToNode gửi một tin nhắn đến một node qua kết nối TCP.
func sendToNode(serverAddress string, message string) (string, error) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return "", fmt.Errorf("error connecting to node: %w", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, message+"\n"); err != nil {
		return "", fmt.Errorf("could not send message: %w", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("lost connection to server: %w", err)
	}
	return response, nil
}
