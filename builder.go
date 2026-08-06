package main

import (
	"embed"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed cmd internal pkg examples go.mod go.sum
var sourceCode embed.FS

func main() {
	app := cli.NewCLI()
	config.SetSourceCode(sourceCode)
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Join the current directory with the config file name
	configPath := filepath.Join(cwd, config.ConfigFileName)

	// --- COMMAND 1: GENERATE DOCKER ARTIFACTS ---
	app.AddCommand("generate gen", "Generate Dockerfile & Compose configs", func(args []string) error {
		if err := generateArtifacts(configPath); err != nil {
			return err
		}

		fmt.Printf("\nDONE! Files created in the 'build/' directory\n")
		fmt.Println("- To start all nodes: khoai start all")
		fmt.Println("- To start a single node: khoai start <node_name>")
		return nil
	})

	// Build exe files
	app.AddCommand("build b", "Build the khoai-node binary into the 'build/' directory", func(args []string) error {
		targetDir := "build"

		fmt.Printf("Building khoai-node binary into './%s' directory...\n", targetDir)

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

	// --- COMMAND: START NODE(S) ---
	app.AddCommand("start", "Build and start node(s) using Docker Compose", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai start node_vingroup | khoai start all")
		}
		nodeToStart := args[0]

		// Always run generate first to ensure build files are up-to-date
		fmt.Println("Checking and generating Docker configuration files...")
		if err := generateArtifacts(configPath); err != nil {
			return fmt.Errorf("could not generate configuration files: %w", err)
		}
		fmt.Println("Docker configuration has been created/updated.")

		composeFile := filepath.Join("build", "docker-compose.yaml")

		if nodeToStart == "all" {
			fmt.Println("\nStarting all nodes...")
			return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d")
		}

		// Validate that the node name exists in the config
		if err := validateNodeName(configPath, nodeToStart); err != nil {
			return err
		}

		fmt.Printf("\nStarting node: %s...\n", nodeToStart)
		return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d", nodeToStart)
	})

	// --- COMMAND: STOP NODE(S) ---
	app.AddCommand("stop", "Stop and remove node container(s)", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai stop node_vingroup | khoai stop all")
		}
		nodeToStop := args[0]
		composeFile := filepath.Join("build", "docker-compose.yaml")

		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			return fmt.Errorf("file 'build/docker-compose.yaml' not found. Please run 'khoai start' first")
		}

		if nodeToStop == "all" {
			fmt.Println("\nStopping all nodes...")
			return runCommand("docker", "compose", "-f", composeFile, "down", "--volumes")
		}

		// Validate that the node name exists in the config
		if err := validateNodeName(configPath, nodeToStop); err != nil {
			return err
		}

		fmt.Printf("\nStopping node: %s...\n", nodeToStop)
		// Stop and remove the container for a specific service
		return runCommand("docker", "compose", "-f", composeFile, "rm", "-s", "-f", "-v", nodeToStop)
	})

	// --- COMMAND: LOGS ---
	app.AddCommand("logs", "View output logs from a running node", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name is required. Example: khoai logs node_vingroup")
		}
		nodeToLog := args[0]
		composeFile := filepath.Join("build", "docker-compose.yaml")

		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			return fmt.Errorf("file 'build/docker-compose.yaml' not found. Please run 'khoai start' first")
		}

		// Validate that the node name exists in the config
		if err := validateNodeName(configPath, nodeToLog); err != nil {
			return err
		}

		fmt.Printf("\nViewing logs for node: %s (Press Ctrl+C to stop)...\n", nodeToLog)
		return runCommand("docker", "compose", "-f", composeFile, "logs", "-f", "--tail", "100", nodeToLog)
	})

	app.Run()
}

// generateArtifacts extracts the logic from the original "generate" command.
func generateArtifacts(configPath string) error {
	// 1. Read YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("could not read file %s: %w", config.ConfigFileName, err)
	}

	var netConf config.NetworkConfig
	if err := yaml.Unmarshal(data, &netConf); err != nil {
		return fmt.Errorf("error parsing YAML file: %w", err)
	}

	// 2. Create build artifacts directory
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}

	// 3. Generate Dockerfile & Config for each Node
	for _, node := range netConf.Nodes {
		nodeDir := filepath.Join(buildDir, node.Name)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return err
		}
		if err := config.GenerateNodeArtifacts(nodeDir, node, netConf); err != nil {
			return fmt.Errorf("error creating files for node %s: %w", node.Name, err)
		}
	}

	// 4. Generate the main docker-compose.yaml
	if err := config.GenerateDockerCompose(buildDir, netConf); err != nil {
		return fmt.Errorf("error creating docker-compose.yaml file: %w", err)
	}

	return nil
}

// runCommand is a helper to execute shell commands and stream output.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Executing: %s\n", cmd.String())
	return cmd.Run()
}

// validateNodeName checks if a given node name exists in the config file.
func validateNodeName(configPath, nodeName string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("could not read config file to validate node name: %w", err)
	}
	var netConf config.NetworkConfig
	if err := yaml.Unmarshal(data, &netConf); err != nil {
		return err
	}
	for _, node := range netConf.Nodes {
		if node.Name == nodeName {
			return nil // Found
		}
	}
	return fmt.Errorf("node '%s' is not defined in file %s", nodeName, config.ConfigFileName)
}
