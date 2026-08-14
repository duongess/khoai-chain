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

	app.AddCommand("join", "Request for a source node to join a target node", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Source and target P2P endpoints are required. Example: khoai join :8080 :8082")
		}
		sourceP2P_onHost := normalizeNodeAddress(args[0])
		targetP2P_onHost := normalizeNodeAddress(args[1])

		// Resolve the source's host endpoint to its internal Docker network address.
		sourceP2P_inDocker, err := findNodeInternalAddressByEndpoint(configPath, sourceP2P_onHost)
		if err != nil {
			return fmt.Errorf("could not resolve source endpoint %s: %w", sourceP2P_onHost, err)
		}

		// The API call goes to the TARGET node's HTTP endpoint.
		httpEndpoint := p2pToHTTP(targetP2P_onHost)

		fmt.Printf("Contacting TARGET node's control plane (%s)\n", httpEndpoint)
		fmt.Printf(" > Requesting that SOURCE node (%s) be allowed to join.\n", sourceP2P_inDocker)

		err = peerAPI(httpEndpoint, "POST", "/join", map[string]string{"source_endpoint": sourceP2P_inDocker}, nil)
		if err != nil {
			return err
		}
		fmt.Printf("\nJoin request sent successfully.\n")
		fmt.Printf("To approve this request, run: khoai approve %s %s\n", targetP2P_onHost, sourceP2P_onHost)
		return nil
	})

	app.AddCommand("requests", "List pending join requests on a node", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Target node address is required. Example: khoai requests :8082")
		}
		targetP2P := normalizeNodeAddress(args[0])
		targetHTTP := p2pToHTTP(targetP2P)
		return peerAPI(targetHTTP, "GET", "/join-requests", nil, nil)
	})

	app.AddCommand("approve", "Approve a pending join request from a source node on a target node", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Target and source node addresses are required. Example: khoai approve :8082 :8080")
		}
		targetP2P := normalizeNodeAddress(args[0])
		sourceNode_onHost := normalizeNodeAddress(args[1])

		sourceNode_inDocker, err := findNodeInternalAddressByEndpoint(configPath, sourceNode_onHost)
		if err != nil {
			return fmt.Errorf("could not resolve source endpoint %s: %w", sourceNode_onHost, err)
		}

		targetHTTP := p2pToHTTP(targetP2P)

		fmt.Printf("Sending approval to node at %s for source node %s to join...\n", targetHTTP, sourceNode_inDocker)
		return peerAPI(targetHTTP, "POST", "/approve", map[string]string{"source_endpoint": sourceNode_inDocker}, nil)
	})

	app.AddCommand("leave", "Leave through the node HTTP control API", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai leave localhost:8080")
		}
		p2pAddress := normalizeNodeAddress(args[0])
		httpAddress := p2pToHTTP(p2pAddress)
		return peerAPI(httpAddress, "POST", "/leave", nil, nil)
	})

	app.AddCommand("remove-peer", "Remove a peer through the HTTP control API", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Target node's P2P address and peer's P2P address are required. Example: khoai remove-peer :8080 :8081")
		}
		targetHTTP := p2pToHTTP(normalizeNodeAddress(args[0]))
		// The body of the request needs the internal P2P address of the peer to remove.
		peerToRemove_inDocker, err := findNodeInternalAddressByEndpoint(configPath, normalizeNodeAddress(args[1]))
		if err != nil {
			return fmt.Errorf("could not resolve peer to remove endpoint %s: %w", args[1], err)
		}
		return peerAPI(targetHTTP, "POST", "/peers/remove", map[string]string{"endpoint": peerToRemove_inDocker}, nil)
	})

	app.AddCommand("peers", "List peers through the node HTTP control API", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai peers localhost:8080")
		}
		p2pAddress := normalizeNodeAddress(args[0])
		httpAddress := p2pToHTTP(p2pAddress)
		return peerAPI(httpAddress, "GET", "/peers", nil, nil)
	})
}

func normalizeNodeAddress(address string) string {
	if len(address) > 0 && address[0] == ':' {
		return "localhost" + address
	}
	return address
}
