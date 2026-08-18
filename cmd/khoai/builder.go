package main

import (
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
)

func main() {
	app := cli.NewCLI()
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Đăng ký các lệnh cơ bản
	// help command
	app.AddCommand("help h", "Show help information", func(args []string) error {
		app.PrintHelp()
		return nil
	}, "General Commands: ")
	// version command
	app.AddCommand("version v", "Display version information", func(args []string) error {
		fmt.Println("Khoai-chain CLI version 1.0.0")
		fmt.Println("See more at: https://github.com/duongess/khoaichain-sdk")
		return nil
	}, "General Commands: ")

	// Lấy đường dẫn config
	configPath := filepath.Join(cwd, config.ConfigFileName)

	// Đăng ký các nhóm lệnh từ các file khác
	registerWorkspaceCommands(app, configPath, "Workspace Management Commands: ")
	registerOrgCommands(app, configPath, "Organization Management Commands: ")
	registerNodeCommands(app, configPath, "Node Management Commands: ")
	registerPeerCommands(app, configPath, "Peer Management Commands: ")

	app.Run()
}
