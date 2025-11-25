package main

import (
	"embed"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed cmd internal pkg examples go.mod go.sum .env
var sourceCode embed.FS

func main() {
	app := cli.NewCLI()
	config.SetSourceCode(sourceCode)

	// --- LỆNH 1: GENERATE DOCKER ARTIFACTS ---
	app.AddCommand("generate gen", "Generate Dockerfile & Compose configs", func(args []string) error {
		fmt.Println("🏭 Đang đọc cấu hình khoai-config.yaml...")

		// 1. Đọc file YAML
		data, err := os.ReadFile("khoai-config.yaml")
		if err != nil {
			return fmt.Errorf("không tìm thấy khoai-config.yaml: %v", err)
		}

		var netConf config.NetworkConfig
		if err := yaml.Unmarshal(data, &netConf); err != nil {
			return err
		}

		// 2. Tạo thư mục build artifacts
		buildDir := "build"
		err = os.MkdirAll(buildDir, 0755)
		if err != nil {
			return err
		}

		// 3. Sinh Dockerfile & Config cho từng Node
		for _, node := range netConf.Nodes {
			err := config.GenerateNodeArtifacts(buildDir, node, netConf)
			if err != nil {
				return err
			}
		}

		// 4. Sinh docker-compose.yaml tổng
		err = config.GenerateDockerCompose(buildDir, netConf)
		if err != nil {
			return err
		}

		fmt.Printf("\n✅ XONG! File được tạo tại thư mục '%s/'\n", buildDir)
		fmt.Println("👉 Để build image: docker compose -f build_artifacts/docker-compose.yaml build")
		fmt.Println("👉 Để chạy test:   docker compose -f build_artifacts/docker-compose.yaml up -d")
		return nil
	})

	// build ra các file exe
	// app.AddCommand("build b", "Build blockchain node (Default)", func(args []string) error {

	// })

	// ... (Giữ lại các lệnh build/clean cũ nếu muốn) ...

	app.Run()
}
