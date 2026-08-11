package main

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/internal/p2p"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
	"strings"
)

const legacyMsgListPeers = "LIST_PEERS"

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

	app.AddCommand("join", "Join a network through an existing peer", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Node and bootstrap addresses are required. Example: khoai join <localhost:8000> <localhost:9000>")
		}
		address := args[0]
		peerAddress := args[1]
		message, err := json.Marshal(p2p.JoinNetworkRequest{
			Type:      p2p.MsgJoinNetwork,
			Address:   address,
			Bootstrap: peerAddress,
		})
		if err != nil {
			return err
		}
		return printNodeResponse(address, string(message))
	})

	app.AddCommand("leave", "Leave the current P2P network", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai leave <localhost:8000>")
		}
		address := args[0]
		message, err := json.Marshal(p2p.LeaveNetworkMessage{Type: p2p.MsgLeaveNetwork, Address: address})
		if err != nil {
			return err
		}
		return printNodeResponse(address, string(message))
	})

	app.AddCommand("approve", "Approve a pending network join request", func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("invalid command. Node address and request ID are required. Example: khoai approve <localhost:9000> <request-id>")
		}
		message, err := json.Marshal(p2p.AcceptJoinMessage{Type: p2p.MsgAcceptJoin, RequestID: args[1]})
		if err != nil {
			return err
		}
		return printNodeResponse(args[0], string(message))
	})

	app.AddCommand("peers", "List peers known by a node", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("invalid command. Node address is required. Example: khoai peers <localhost:8000>")
		}
		address := args[0]
		message, err := json.Marshal(p2p.PeerListMessage{Type: p2p.MsgPeerList, Request: true})
		if err != nil {
			return err
		}
		response, err := sendToNode(address, string(message))
		if err != nil {
			return err
		}
		// Older released node images called this request LIST_PEERS. Keep the
		// CLI usable while the containers are being rebuilt with PEER_LIST.
		if isUnknownCommandResponse(response) {
			legacy, err := json.Marshal(p2p.PeerListMessage{Type: legacyMsgListPeers, Request: true})
			if err != nil {
				return err
			}
			response, err = sendToNode(address, string(legacy))
			if err != nil {
				return err
			}
		}
		fmt.Print(response)
		return nil
	})
}

func printNodeResponse(address, message string) error {
	response, err := sendToNode(address, message)
	if err != nil {
		return err
	}
	fmt.Print(response)
	return nil
}

func isUnknownCommandResponse(response string) bool {
	var result p2p.ResponseMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &result); err != nil {
		return false
	}
	return strings.EqualFold(result.Status, "Error") && strings.Contains(strings.ToLower(result.Error), "unknown command")
}
