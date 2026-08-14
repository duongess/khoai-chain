package main

import (
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
)

// registerNodeCommands đăng ký các lệnh tương tác với node/container.
func registerNodeCommands(app *cli.CLI, configPath string) {
	// Lệnh 'start' để build và khởi động node(s) bằng Docker Compose.
	app.AddCommand("start", "Build and start node(s) using Docker Compose", func(args []string) error {
		isWorkspace, err := isWorkspaceContext()
		if err != nil {
			return err
		}

		nodeToStart := ""
		if len(args) > 0 {
			nodeToStart = args[0]
		}
		if !isWorkspace && nodeToStart == "" {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai start node_vingroup | khoai start all")
		}

		var composeFile string
		if isWorkspace {
			fmt.Println("Running in Organization Workspace context.")
			fmt.Println("Preparing node artifacts for this organization...")
			if _, err := generateWorkspaceNodeArtifacts(true); err != nil {
				return err
			}
			composeFile = "docker-compose.yaml"
			if err := generateWorkspaceCompose(composeFile); err != nil {
				return fmt.Errorf("could not generate workspace docker-compose: %w", err)
			}
		} else {
			fmt.Println("Running in Builder context.")
			if err := generateArtifacts(configPath); err != nil {
				return fmt.Errorf("could not generate configuration files: %w", err)
			}
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if isWorkspace {
			if nodeToStart == "all" {
				fmt.Println("\nStarting all nodes...")
			} else {
				fmt.Println("\nStarting all nodes in this organization...")
			}
			return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d", "--remove-orphans")
		}

		if nodeToStart == "all" {
			fmt.Println("\nStarting all nodes...")
			return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d", "--remove-orphans")
		}

		fmt.Printf("\nStarting node: %s...\n", nodeToStart)
		return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d", "--remove-orphans", nodeToStart)
	})

	// Lệnh 'stop' để dừng và xóa container của node(s).
	app.AddCommand("stop", "Stop and remove node container(s)", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai stop node_vingroup | khoai stop all")
		}
		nodeToStop := args[0]

		isWorkspace, _ := isWorkspaceContext()
		var composeFile string
		if isWorkspace {
			composeFile = "docker-compose.yaml"
		} else {
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not found. Please run 'khoai start' first", composeFile)
		}

		if nodeToStop == "all" {
			fmt.Println("\nStopping all nodes...")
			return runCommand("docker", "compose", "-f", composeFile, "down", "--volumes")
		}

		fmt.Printf("\nStopping node: %s...\n", nodeToStop)
		return runCommand("docker", "compose", "-f", composeFile, "rm", "-s", "-f", "-v", nodeToStop)
	})

	// Lệnh 'logs' để xem log từ một node đang chạy.
	app.AddCommand("logs", "View output logs from a running node", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name is required. Example: khoai logs node_vingroup")
		}
		nodeToLog := args[0]

		isWorkspace, _ := isWorkspaceContext()
		var composeFile string
		if isWorkspace {
			composeFile = "docker-compose.yaml"
		} else {
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not found. Please run 'khoai start' first", composeFile)
		}

		fmt.Printf("\nViewing logs for node: %s (Press Ctrl+C to stop)...\n", nodeToLog)
		return runCommand("docker", "compose", "-f", composeFile, "logs", "-f", "--tail", "100", nodeToLog)
	})

	app.AddCommand("join", "Connect a source node to a target node", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Source and target P2P endpoints are required. Example: khoai join :8080 :8082")
		}
		sourceP2P := normalizeNodeAddress(args[0])
		targetP2P := normalizeNodeAddress(args[1])

		// The API call goes to the source node's HTTP endpoint.
		httpEndpoint := p2pToHTTP(sourceP2P)

		fmt.Printf("Sending join request to %s for node %s to connect to %s\n", httpEndpoint, sourceP2P, targetP2P)

		return peerAPI(httpEndpoint, "POST", "/join", map[string]string{"target": targetP2P})
	})

	app.AddCommand("leave", "Leave through the node HTTP control API", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai leave localhost:8080")
		}
		address := args[0]
		address = normalizeNodeAddress(address)
		return peerAPI(address, "POST", "/leave", nil)
	})

	app.AddCommand("remove-peer", "Remove a peer through the HTTP control API", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. HTTP node address and P2P peer endpoint are required. Example: khoai remove-peer localhost:8080 localhost:8081")
		}
		return peerAPI(normalizeNodeAddress(args[0]), "POST", "/peers/remove", map[string]string{"endpoint": args[1]})
	})

	app.AddCommand("peers", "List peers through the node HTTP control API", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai peers localhost:8080")
		}
		address := args[0]
		address = normalizeNodeAddress(address)
		return peerAPI(address, "GET", "/peers", nil)
	})
}

func normalizeNodeAddress(address string) string {
	if len(address) > 0 && address[0] == ':' {
		return "localhost" + address
	}
	return address
}
