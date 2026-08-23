package main

import (
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
)

// registerNodeCommands đăng ký các lệnh tương tác với node/container.
func registerNodeCommands(app *cli.CLI, configPath string, groupName string) {
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
		if !isWorkspace {
			fmt.Println("Running in Organization Workspace context.")
			composeFile = "docker-compose.yaml"
		} else {
			fmt.Println("Running in Builder context.")
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if !isWorkspace {
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
	}, groupName)

	// Lệnh 'stop' để dừng và xóa container của node(s).
	app.AddCommand("stop", "Stop and remove node container(s)", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai stop node_vingroup | khoai stop all")
		}
		nodeToStop := args[0]

		isWorkspace, err := isWorkspaceContext()
		if err != nil {
			return fmt.Errorf(err.Error())
		}
		var composeFile string
		if !isWorkspace {
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
	}, groupName)

	// Lệnh 'logs' để xem log từ một node đang chạy.
	app.AddCommand("logs", "View output logs from a running node", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name is required. Example: khoai logs node_vingroup")
		}
		nodeToLog := args[0]

		isWorkspace, err := isWorkspaceContext()
		if err != nil {
			return fmt.Errorf(err.Error())
		}
		var composeFile string
		if !isWorkspace {
			composeFile = "docker-compose.yaml"
		} else {
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not found. Please run 'khoai start' first", composeFile)
		}

		fmt.Printf("\nViewing logs for node: %s (Press Ctrl+C to stop)...\n", nodeToLog)
		return runCommand("docker", "compose", "-f", composeFile, "logs", "-f", "--tail", "100", nodeToLog)
	}, groupName)
}
