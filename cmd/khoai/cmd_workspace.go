package main

import (
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
)

// registerWorkspaceCommands đăng ký các lệnh CLI
func registerWorkspaceCommands(app *cli.CLI, configPath string, groupName string) {

	// 1. Lệnh INIT: Chỉ tập trung sinh file cấu hình và thư mục, KHÔNG tải mã nguồn
	app.AddCommand("init i", "Initializes a new Khoai workspace", func(args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
			fmt.Printf("Initializing new Khoai workspace in '%s'...\n", targetDir)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}
		} else {
			fmt.Println("Initializing new Khoai workspace in the current directory...")
		}

		if err := os.MkdirAll(filepath.Join(targetDir, "nodes"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(targetDir, "contracts"), 0755); err != nil {
			return err
		}

		if err := createDefaultWorkspaceFiles(targetDir); err != nil {
			return fmt.Errorf("failed to create default workspace files: %w", err)
		}

		fmt.Println("Workspace initialized successfully. You can now add your smart contracts to the 'contracts/' directory and run 'khoai build'.")
		return nil
	}, groupName)

	// 2. Lệnh BUILD: Gộp chung logic của 'gen' và 'build' cũ. Tự nhận diện ngữ cảnh.
	app.AddCommand("build b", "Download source and generate artifacts based on config", func(args []string) error {
		isMultiOrg, err := isWorkspaceOrganizations()
		if err != nil {
			return err
		}

		fmt.Println("Downloading latest Khoai source code...")
		// Nếu là đa tổ chức, tải source vào build/. Nếu đơn tổ chức, tải vào root (.)
		targetBuildDir := "."
		if isMultiOrg {
			targetBuildDir = config.BuildDir
		}

		version, err := downloadViaScript("latest", targetBuildDir)
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded version %s.\n", version)

		fmt.Println("Generating node artifacts...")
		if isMultiOrg {
			// Case: Đa tổ chức (Tương đương lệnh 'gen' cũ)
			if err := generateArtifacts(configPath); err != nil {
				return err
			}
			fmt.Printf("\nDONE! Artifacts generated in '%s/' directory.\n", config.BuildDir)
		} else {
			// Case: Đơn tổ chức (Workspace)
			nodesGenerated, err := generateWorkspaceNodeArtifacts(false)
			if err != nil {
				return err
			}
			if nodesGenerated > 0 {
				fmt.Printf("\nSuccessfully generated artifacts for %d new node(s).\n", nodesGenerated)
			} else {
				fmt.Println("\nAll nodes are up-to-date. No new artifacts were generated.")
			}
		}

		return nil
	}, groupName)

	// 3. Lệnh UPDATE: Cập nhật source (Đã sửa lỗi trùng tên biến isWorkspaceOrganizations gây shadow)
	app.AddCommand("update u", "Update Khoai source code version", func(args []string) error {
		isMultiOrg, err := isWorkspaceOrganizations()
		if err != nil {
			return err
		}

		var version string
		if isMultiOrg {
			version, err = downloadViaScript("latest", config.BuildDir)
		} else {
			version, err = downloadViaScript("latest", ".")
		}

		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully updated to version %s.\n", version)
		return nil
	}, groupName)

	app.AddCommand("dev", "Start local web interface for testing contract and peers management", func(args []string) error {
		var targetNode = ":8080"
		isMultiOrg, err := isWorkspaceOrganizations()
		if err != nil {
			return err
		}
		var contractAbi interface{}
		if isMultiOrg {
			contractAbi, err = getContractAbi(config.BuildDir)
		} else {
			contractAbi, err = getContractAbi("./")
		}

		if len(args) > 0 {
			targetNode = args[0]
			fmt.Printf("Start connecting to %v\n", targetNode)
		}
		return cli.RunUI(targetNode, contractAbi)
	}, groupName)
}
