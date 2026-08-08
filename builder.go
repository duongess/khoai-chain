package main

import (
	"bufio"
	"bytes"
	"embed"
	"flag"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	app.AddCommand("generate gen", "Download source and generate Docker configs", func(args []string) error {
		// Create a flag set for this command
		genFlags := flag.NewFlagSet("generate", flag.ExitOnError)
		versionFlag := genFlags.String("version", "latest", "The source code version to download (e.g., v1.0.1)")

		// Parse the arguments for this command
		if err := genFlags.Parse(args); err != nil {
			return err
		}

		version := *versionFlag

		// 1. Download and extract source code
		fmt.Printf("Downloading source code version: %s...\n", version)
		downloadedVersion, err := downloadViaScript(version)
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded and extracted version %s.\n", downloadedVersion)

		// 2. Generate artifacts
		if err := generateArtifacts(configPath, downloadedVersion); err != nil {
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
		// Pass empty version to skip download-related steps like creating .version file
		if err := generateArtifacts(configPath, ""); err != nil {
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

	app.AddCommand("connect", "Connect to peer", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Both server and peer addresses are required. Example: khoai connect <localhost:8000> <localhost:9000>")
		}
		address := args[0]
		peerAddress := args[1]
		sendToNode(address, fmt.Sprintf("{\"type\":\"CONNECT_PEER\", \"address\":\"%s\"}", peerAddress))
		return nil
	})

	app.Run()
}

// generateArtifacts extracts the logic from the original "generate" command.
func generateArtifacts(configPath string, version string) error {
	// The .version file is now created by the install.sh script during `khoai gen`.

	// 1. Load or create default config
	builderConf, err := config.LoadBuilderConfig(configPath)
	if err != nil {
		return err
	}

	// 2. Create build artifacts directory
	if err := os.MkdirAll(config.NodesBaseDir, 0755); err != nil {
		return err
	}

	// 3. Generate Dockerfile & Config for each Node by iterating through organizations
	for _, org := range builderConf.Organizations {
		for _, node := range org.Nodes {
			uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(org.DisplayName), node.ID)
			nodeDir := filepath.Join(config.NodesBaseDir, uniqueNodeName)
			if err := os.MkdirAll(nodeDir, 0755); err != nil {
				return err
			}
			if err := config.GenerateNodeArtifacts(nodeDir, node, org, builderConf); err != nil {
				return fmt.Errorf("error creating files for node %s: %w", uniqueNodeName, err)
			}
		}
	}

	// 4. Generate the main docker-compose.yaml
	if err := config.GenerateDockerCompose(config.BuildDir, builderConf); err != nil {
		return fmt.Errorf("error creating docker-compose.yaml file: %w", err)
	}

	return nil
}

// downloadAndUnzipSource handles fetching and extracting the project source code from GitHub releases.
func downloadViaScript(version string) (string, error) {
	fmt.Printf("Starting process to download source code version: %s\n", version)

	// Setup command to run the install.sh script with the specified version.
	scriptURL := "https://raw.githubusercontent.com/duongess/khoai-chain/main/install.sh"
	shellCmd := fmt.Sprintf("curl -fsSL %s | bash -s -- %s", scriptURL, version)
	cmd := exec.Command("bash", "-c", shellCmd)

	// Capture stdout to get the version string returned by the script.
	var out bytes.Buffer
	cmd.Stdout = &out
	// Pipe the script's stderr to our stderr to show real-time progress.
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error executing install.sh script: %w", err)
	}

	// The script is expected to print the downloaded version tag to stdout.
	downloadedVersion := strings.TrimSpace(out.String())
	if downloadedVersion == "" {
		return "", fmt.Errorf("install.sh script did not output a version string")
	}
	return downloadedVersion, nil
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
	builderConf, err := config.LoadBuilderConfig(configPath)
	if err != nil {
		// If config loading fails, we can't validate.
		return fmt.Errorf("could not load configuration to validate node name: %w", err)
	}

	for _, org := range builderConf.Organizations {
		for _, node := range org.Nodes {
			uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(org.DisplayName), node.ID)
			if uniqueNodeName == nodeName {
				return nil // Found
			}
		}
	}

	return fmt.Errorf("node '%s' is not defined in file %s", nodeName, config.ConfigFileName)
}

func sanitize(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}

func sendToNode(serverAddress string, message string) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Printf("Error connecting to node: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Fprintf(conn, string(message)+"\n")

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Lost connection to server." + err.Error())
		return
	}

	fmt.Println(response)
}
