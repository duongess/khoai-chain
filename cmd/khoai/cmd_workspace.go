package main

import (
	"fmt"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// registerWorkspaceCommands đăng ký các lệnh liên quan đến workspace như 'init' và 'build'.
func registerWorkspaceCommands(app *cli.CLI, configPath string, groupName string) {

	app.AddCommand("generate gen", "Download source and generate mess", func(args []string) error {
		// 1. Download source code
		fmt.Println("Downloading latest Khoai source code...")
		version, err := downloadViaScript("latest", config.BuildDir)
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded and extracted version %s.\n", version)

		// 2. Generate artifacts
		if err := generateArtifacts(configPath); err != nil {
			return err
		}

		fmt.Printf("\nDONE! Files created in the 'build/' directory\n")
		fmt.Println("- To start all nodes: khoai start all")
		fmt.Println("- To start a single node: khoai start <node_name>")
		return nil
	}, groupName)

	// Lệnh 'init' để khởi tạo một workspace mới.
	app.AddCommand("init i", "Initializes the current directory as a new Khoai organization workspace", func(args []string) error {
		fmt.Println("Initializing new Khoai organization workspace in the current directory...")
		_, err1 := os.Stat(config.ConfigFileName)
		configExists := !os.IsNotExist(err1)

		_, err2 := os.Stat("organization.yaml")
		orgExists := !os.IsNotExist(err2)

		if configExists || orgExists {
			fmt.Println("Found existing workspace files. Configuration files will be overwritten.")
		}

		fmt.Println("Downloading latest Khoai source code...")
		version, err := downloadViaScript("latest", ".")
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded version %s.\n", version)

		fmt.Println("Creating workspace files...")
		if err := os.MkdirAll("nodes", 0755); err != nil {
			return err
		}
		if err := os.MkdirAll("contracts", 0755); err != nil {
			return err
		}

		if err := createDefaultWorkspaceFiles("."); err != nil {
			return fmt.Errorf("failed to create default workspace files: %w", err)
		}

		cwd, _ := os.Getwd()
		fmt.Printf("Successfully initialized workspace for organization '%s'.\n", filepath.Base(cwd))
		return nil
	}, groupName)

	// Lệnh 'build' để tạo các artifacts cho node trong workspace.
	app.AddCommand("build b", "Generate artifacts for new nodes in the workspace", func(args []string) error {
		isWorkspace, err := isWorkspaceContext()
		if err != nil {
			return err
		}
		if !isWorkspace {
			return fmt.Errorf("the 'build' command can only be run inside an initialized workspace (missing organization.yaml)")
		}

		fmt.Println("Building node artifacts in workspace...")

		nodesGenerated, err := generateWorkspaceNodeArtifacts(false)
		if err != nil {
			return err
		}

		if nodesGenerated > 0 {
			fmt.Printf("\nSuccessfully generated artifacts for %d new node(s).\n", nodesGenerated)
		} else {
			fmt.Println("\nAll nodes are up-to-date. No new artifacts were generated.")
		}
		return nil
	}, groupName)

	app.AddCommand("update u", "update version of khoai source code and rebuild artifacts for all nodes in the workspace", func(args []string) error {
		isArtifacts, err := isArtifacts()
		if err != nil {
			return err
		}
		var version string
		if isArtifacts {
			version, err = downloadViaScript("latest", "./build")
		} else {
			version, err = downloadViaScript("latest", ".")
		}

		fmt.Println("Updating Khoai source code to the latest version...")
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully updated to version %s.\n", version)
		return nil
	}, groupName)
}

func generateWorkspaceNodeArtifacts(force bool) (int, error) {
	rootConf, err := config.LoadBuilderConfig(config.ConfigFileName)
	if err != nil {
		return 0, fmt.Errorf("could not load workspace khoai-config.yaml: %w", err)
	}
	orgData, err := os.ReadFile("organization.yaml")
	if err != nil {
		return 0, fmt.Errorf("could not load workspace organization.yaml: %w", err)
	}
	var orgConf config.OrganizationConfig
	if err := yaml.Unmarshal(orgData, &orgConf); err != nil {
		return 0, fmt.Errorf("could not parse workspace organization.yaml: %w", err)
	}

	nodesBaseDir := "nodes"
	nodesGenerated := 0
	for _, node := range orgConf.Nodes {
		nodeDir := filepath.Join(nodesBaseDir, node.ID)

		if _, err := os.Stat(nodeDir); err == nil && !force {
			fmt.Printf("- Node '%s' already exists, skipping.\n", node.ID)
			continue
		}

		fmt.Printf("- Generating artifacts for node: '%s'\n", node.ID)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return 0, fmt.Errorf("could not create directory for node %s: %w", node.ID, err)
		}

		uniqueNodeName := fmt.Sprintf("%s-%s", sanitize(orgConf.DisplayName), node.ID)
		if err := config.GenerateWorkspaceNodeArtifacts(nodeDir, node, orgConf, rootConf, uniqueNodeName); err != nil {
			return 0, fmt.Errorf("error creating files for node %s: %w", node.ID, err)
		}
		nodesGenerated++
	}

	return nodesGenerated, nil
}

// createDefaultWorkspaceFiles tạo các file cấu hình mặc định cho một workspace.
func createDefaultWorkspaceFiles(dir string) error {
	defaultCfg := config.GetDefaultBuilderConfig()
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}
	defaultOrg := config.OrganizationConfig{
		DisplayName: filepath.Base(cwd),
		Nodes: []config.RuntimeNodeConfig{
			{ID: "node1", DisplayName: "Default Node", Endpoint: "localhost:9000"},
		},
	}

	orgYAML, err := yaml.Marshal(defaultOrg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "organization.yaml"), orgYAML, 0644); err != nil {
		return err
	}

	rootCfg := config.BuilderConfig{Network: defaultCfg.Network, Docker: defaultCfg.Docker}
	rootYAML, err := yaml.Marshal(rootCfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, config.ConfigFileName), rootYAML, 0644)
}

// isWorkspaceContext kiểm tra xem thư mục hiện tại có phải là một workspace hay không.
func isWorkspaceContext() (bool, error) {
	if _, err := os.Stat("organization.yaml"); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// generateWorkspaceCompose tạo file docker-compose.yaml trong một workspace.
func generateWorkspaceCompose(composePath string) error {
	rootConf, err := config.LoadBuilderConfig(config.ConfigFileName)
	if err != nil {
		return fmt.Errorf("could not load workspace khoai-config.yaml: %w", err)
	}

	orgData, err := os.ReadFile("organization.yaml")
	if err != nil {
		return fmt.Errorf("could not load workspace organization.yaml: %w", err)
	}
	var orgConf config.OrganizationConfig
	if err := yaml.Unmarshal(orgData, &orgConf); err != nil {
		return fmt.Errorf("could not parse workspace organization.yaml: %w", err)
	}

	workspaceBuilderConfig := &config.BuilderConfig{
		Network:       rootConf.Network,
		Docker:        rootConf.Docker,
		Organizations: []config.OrganizationConfig{orgConf},
	}

	if err := config.GenerateWorkspaceDockerCompose(".", workspaceBuilderConfig); err != nil {
		return err
	}
	fmt.Println("Generated workspace docker-compose.yaml")
	return nil
}
