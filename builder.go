package main

import (
	"embed"
	"fmt"
	"khoai-chain/internal/config"
	utils "khoai-chain/internal/ulits"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed cmd internal pkg examples go.mod go.sum .env
var sourceCode embed.FS

func main() {
	app := cli.NewCLI()
	config.SetSourceCode(sourceCode)
	utils.SetSourceCode(sourceCode)

	// --- LỆNH 1: GENERATE DOCKER ARTIFACTS ---
	app.AddCommand("generate gen", "Generate Dockerfile & Compose configs", func(args []string) error {
		fmt.Println("🏭 Đang đọc cấu hình khoai-config.yaml...")

		// 1. Đọc file YAML
		filePath, err := utils.GetEnv("KHOAI_FILE_CONFIG")
		if err != nil {
			return fmt.Errorf("❌ Lỗi: %v", err)
		}
		data, err := sourceCode.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("không tìm thấy .env: %v", err)
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
			nodeDir := filepath.Join(buildDir, node.Name)
			if err := os.MkdirAll(nodeDir, 0755); err != nil {
				return err
			}
			err := config.GenerateNodeArtifacts(nodeDir, node, netConf)
			if err != nil {
				return err
			}
			err = config.BuildExe(nodeDir)
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
	app.AddCommand("build b", "Build the blockchain node binary from source", func(args []string) error {
		targetDir := "."

		// If user provides a specific path, use it
		if len(args) > 0 {
			targetDir = args[0]
		}

		fmt.Printf("🔨 Starting build process in: %s\n", targetDir)

		err := config.BuildExe(targetDir)
		if err != nil {
			return fmt.Errorf("failed to build node: %v", err)
		}

		return nil
	})

	// lệnh help
	app.AddCommand("help h", "Show help information", func(args []string) error {
		app.PrintHelp()
		return nil
	})

	// lenh version
	app.AddCommand("version v", "Display version information", func(args []string) error {
		fmt.Println("Khoai-chain CLI version 1.0.0")
		fmt.Println("See more at: https://github.com/duongess/khoaichain-sdk")
		return nil
	})

	app.Run()
}
