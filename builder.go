package main

import (
	"embed"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed cmd internal pkg examples go.mod go.sum
var sourceCode embed.FS

func main() {
	app := cli.NewCLI()
	config.SetSourceCode(sourceCode)

	// --- COMMAND 1: GENERATE DOCKER ARTIFACTS ---
	app.AddCommand("generate gen", "Generate Dockerfile & Compose configs", func(args []string) error {
		fmt.Println("Reading khoai-config.yaml configuration...")

		// 1. Read YAML file
		filePath := config.ConfigFileName
		data, err := sourceCode.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("Error reading %s: %v", filePath, err)
		}

		var netConf config.NetworkConfig
		if err := yaml.Unmarshal(data, &netConf); err != nil {
			return err
		}

		// 2. Create build artifacts directory
		buildDir := "build"
		err = os.MkdirAll(buildDir, 0755)
		if err != nil {
			return err
		}

		// 3. Generate Dockerfile & Config for each Node
		for _, node := range netConf.Nodes {
			nodeDir := filepath.Join(buildDir, node.Name)
			if err := os.MkdirAll(nodeDir, 0755); err != nil {
				return err
			}
			err := config.GenerateNodeArtifacts(nodeDir, node, netConf)
			if err != nil {
				return err
			}
		}

		// 4. Generate the main docker-compose.yaml
		err = config.GenerateDockerCompose(buildDir, netConf)
		if err != nil {
			return err
		}

		fmt.Printf("\nDONE! Files created in the '%s/' directory\n", buildDir)
		fmt.Println("- To build images: docker compose -f build/docker-compose.yaml build")
		fmt.Println("- To run:   docker compose -f build/docker-compose.yaml up -d")
		return nil
	})

	// Build exe files
	app.AddCommand("build b", "Build the blockchain node binary", func(args []string) error {
		targetDir := "."

		// If user provides a specific path, use it
		if len(args) > 0 {
			targetDir = args[0]
		}

		fmt.Printf("Starting build process in: %s\n", targetDir)

		err := config.BuildExe(targetDir)
		if err != nil {
			return fmt.Errorf("failed to build node: %v", err)
		}

		return nil
	})

	// help command
	app.AddCommand("help h", "Show help information", func(args []string) error {
		app.PrintHelp()
		return nil
	})

	// version command
	app.AddCommand("version v", "Display version information", func(args []string) error {
		fmt.Println("Khoai-chain CLI version 1.0.0")
		fmt.Println("See more at: https://github.com/duongess/khoaichain-sdk")
		return nil
	})

	app.Run()
}
