package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"embed"
	"flag"
	"fmt"
	"io"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		downloadedVersion, err := downloadViaScript(version, config.BuildDir)
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded and extracted version %s.\n", downloadedVersion)

		// 2. Generate artifacts
		if err := generateArtifacts(configPath); err != nil {
			return err
		}

		fmt.Printf("\nDONE! Files created in the 'build/' directory\n")
		fmt.Println("- To start all nodes: khoai start all")
		fmt.Println("- To start a single node: khoai start <node_name>")
		return nil
	})

	app.AddCommand("init i", "Init a organization", func(args []string) error {
		fmt.Println("Initializing organization...")

		// Create a flag set for this command
		genFlags := flag.NewFlagSet("init", flag.ExitOnError)
		versionFlag := genFlags.String("version", "latest", "The source code version to download (e.g., v1.0.1)")

		// Parse the arguments for this command
		if err := genFlags.Parse(args); err != nil {
			return err
		}

		version := *versionFlag

		// 1. Download and extract source code
		fmt.Printf("Downloading source code version: %s...\n", version)
		downloadedVersion, err := downloadViaScript(version, "./")
		if err != nil {
			return fmt.Errorf("failed to download source code: %w", err)
		}
		fmt.Printf("Successfully downloaded and extracted version %s.\n", downloadedVersion)

		fmt.Println("Organization initialized successfully.")
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

	// --- COMMAND: ORG ---
	app.AddCommand("org", "Manage organizations", func(args []string) error {
		if len(args) < 2 {
			fmt.Println("Invalid command. Use 'org init <name>' or 'org package <name>'.")
			return nil
		}
		action, name := args[0], args[1]
		switch action {
		case "init":
			return initOrganization(name)
		case "package":
			return packageOrganization(name, configPath)
		default:
			return fmt.Errorf("unknown org command: %s", action)
		}
	})

	// --- COMMAND: PACKAGE (shortcut for org package) ---
	app.AddCommand("package", "Package an organization (e.g., package org <name>)", func(args []string) error {
		if len(args) < 2 || args[0] != "org" {
			return fmt.Errorf("invalid command. Use 'package org <name>'")
		}
		return packageOrganization(args[1], configPath)
	})

	// --- COMMAND: INSTALL ---
	app.AddCommand("install", "Install an organization from a package file", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("package file path is required")
		}
		packagePath := args[0]
		return installOrganization(packagePath)
	})

	// --- COMMAND: START NODE(S) ---
	app.AddCommand("start", "Build and start node(s) using Docker Compose", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("invalid command. A node name or 'all' is required. Example: khoai start node_vingroup | khoai start all")
		}
		nodeToStart := args[0]

		// Check if running in a workspace
		isWorkspace, err := isWorkspaceContext()
		if err != nil {
			return err
		}

		var composeFile string
		if isWorkspace {
			fmt.Println("Running in Organization Workspace context.")
			composeFile = "docker-compose.yaml"
			if err := generateWorkspaceCompose(composeFile); err != nil {
				return fmt.Errorf("could not generate workspace docker-compose: %w", err)
			}
		} else {
			fmt.Println("Running in Builder context.")
			// Always run generate first to ensure build files are up-to-date
			if err := generateArtifacts(configPath); err != nil {
				return fmt.Errorf("could not generate configuration files: %w", err)
			}
			composeFile = filepath.Join(config.BuildDir, "docker-compose.yaml")
		}

		if nodeToStart == "all" {
			fmt.Println("\nStarting all nodes...")
			return runCommand("docker", "compose", "-f", composeFile, "up", "--build", "-d")
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
		// Stop and remove the container for a specific service
		return runCommand("docker", "compose", "-f", composeFile, "rm", "-s", "-f", "-v", nodeToStop)
	})

	// --- COMMAND: LOGS ---
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

// generateArtifacts reads the main builder config and generates artifacts for all organizations.
func generateArtifacts(configPath string) error {
	fmt.Println("Generating build artifacts...")
	// 1. Load or create default config
	builderConf, err := config.LoadBuilderConfig(configPath)
	if err != nil {
		return err
	}

	// 2. Define base directory for organization artifacts
	orgsBaseDir := filepath.Join(config.BuildDir, config.OrgsDir)
	if err := os.MkdirAll(orgsBaseDir, 0755); err != nil {
		return err
	}

	// 3. Generate artifacts for each organization
	for _, org := range builderConf.Organizations {
		fmt.Printf("- Generating artifacts for organization: %s\n", org.DisplayName)
		orgDir := filepath.Join(orgsBaseDir, sanitize(org.DisplayName))
		if err := config.GenerateOrganizationArtifacts(orgDir, org, builderConf); err != nil {
			return fmt.Errorf("error generating artifacts for organization %s: %w", org.DisplayName, err)
		}
	}

	// 4. Generate the main docker-compose.yaml
	fmt.Println("- Generating main docker-compose.yaml")
	if err := config.GenerateDockerCompose(config.BuildDir, builderConf); err != nil {
		return fmt.Errorf("error creating docker-compose.yaml file: %w", err)
	}

	fmt.Println("Artifact generation complete.")
	return nil
}

// initOrganization creates a new, empty organization workspace.
func initOrganization(orgName string) error {
	fmt.Printf("Initializing new organization workspace: %s\n", orgName)
	sanitizedName := sanitize(orgName)
	if _, err := os.Stat(sanitizedName); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", sanitizedName)
	}

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(sanitizedName, "nodes"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(sanitizedName, "contracts"), 0755); err != nil {
		return err
	}

	// Create default config files
	defaultCfg := config.GetDefaultBuilderConfig()
	defaultCfg.Organizations[0].DisplayName = orgName

	// organization.yaml
	orgYAML, _ := yaml.Marshal(defaultCfg.Organizations[0])
	if err := os.WriteFile(filepath.Join(sanitizedName, "organization.yaml"), orgYAML, 0644); err != nil {
		return err
	}

	// khoai.yaml
	rootCfg := config.BuilderConfig{Network: defaultCfg.Network, Docker: defaultCfg.Docker}
	rootYAML, _ := yaml.Marshal(rootCfg)
	if err := os.WriteFile(filepath.Join(sanitizedName, config.ConfigFileName), rootYAML, 0644); err != nil {
		return err
	}

	fmt.Printf("Successfully created organization workspace in './%s/'\n", sanitizedName)
	return nil
}

// packageOrganization creates a .tar.gz archive of a generated organization.
func packageOrganization(orgName, configPath string) error {
	fmt.Printf("Packaging organization: %s\n", orgName)

	// Ensure artifacts are generated first
	if err := generateArtifacts(configPath); err != nil {
		return fmt.Errorf("failed to generate artifacts before packaging: %w", err)
	}

	sanitizedName := sanitize(orgName)
	orgSrcDir := filepath.Join(config.BuildDir, config.OrgsDir, sanitizedName)
	if _, err := os.Stat(orgSrcDir); os.IsNotExist(err) {
		return fmt.Errorf("organization '%s' not found in build directory. Have you run 'khoai generate'?", orgName)
	}

	// Create dist dir
	if err := os.MkdirAll(config.DistDir, 0755); err != nil {
		return err
	}

	// Create package file
	// TODO: Use a real version
	version := "1.0.0"
	packageName := fmt.Sprintf("%s-khoai-v%s.tar.gz", sanitizedName, version)
	packagePath := filepath.Join(config.DistDir, packageName)
	file, err := os.Create(packagePath)
	if err != nil {
		return fmt.Errorf("could not create package file: %w", err)
	}
	defer file.Close()

	// Setup tar.gz writer
	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Walk the organization source directory and add files to tar
	return filepath.Walk(orgSrcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == orgSrcDir {
			return nil // Skip root dir
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Set name to be relative to the org dir, inside a root folder
		relPath, err := filepath.Rel(orgSrcDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(sanitizedName, relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// installOrganization extracts a package into a new workspace.
func installOrganization(packagePath string) error {
	fmt.Printf("Installing organization from: %s\n", packagePath)

	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return fmt.Errorf("could not get absolute path for package: %w", err)
	}

	installDir := filepath.Dir(absPath)

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("could not open package: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// First, determine the root directory name and check if it exists
	header, err := tr.Next()
	if err == io.EOF {
		return fmt.Errorf("package is empty")
	}
	orgDirName := strings.Split(header.Name, "/")[0]
	targetDir := filepath.Join(installDir, orgDirName)

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		return fmt.Errorf("installation directory '%s' already exists", targetDir)
	}

	// Reset reader to process all files
	file.Seek(0, 0)
	gr.Reset(file)
	tr = tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Path traversal check
		target := filepath.Join(installDir, header.Name)
		if !strings.HasPrefix(target, installDir) {
			return fmt.Errorf("invalid path in package: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	fmt.Printf("Successfully installed organization to '%s'\n", targetDir)
	return nil
}

// downloadAndUnzipSource handles fetching and extracting the project source code from GitHub releases.
func downloadViaScript(version string, targetDir string) (string, error) {
	fmt.Printf("Starting process to download source code version: %s\n", version)

	// Setup command to run the install.sh script with the specified version.
	scriptURL := "https://raw.githubusercontent.com/duongess/khoai-chain/main/install.sh"
	shellCmd := fmt.Sprintf("curl -fsSL %s | bash -s -- %s %s", scriptURL, version, targetDir)
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

// isWorkspaceContext checks if the current directory is an organization workspace.
func isWorkspaceContext() (bool, error) {
	if _, err := os.Stat("organization.yaml"); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// generateWorkspaceCompose generates a docker-compose.yaml file within a workspace.
func generateWorkspaceCompose(composePath string) error {
	// Load khoai.yaml for network/docker settings
	rootConf, err := config.LoadBuilderConfig(config.ConfigFileName)
	if err != nil {
		return fmt.Errorf("could not load workspace khoai.yaml: %w", err)
	}

	// Load organization.yaml for org/node settings
	orgData, err := os.ReadFile("organization.yaml")
	if err != nil {
		return fmt.Errorf("could not load workspace organization.yaml: %w", err)
	}
	var orgConf config.OrganizationConfig
	if err := yaml.Unmarshal(orgData, &orgConf); err != nil {
		return fmt.Errorf("could not parse workspace organization.yaml: %w", err)
	}

	// Combine into a single BuilderConfig for the generator function
	workspaceBuilderConfig := &config.BuilderConfig{
		Network:       rootConf.Network,
		Docker:        rootConf.Docker,
		Organizations: []config.OrganizationConfig{orgConf},
	}

	// Generate the compose file in the current directory (".")
	if err := config.GenerateWorkspaceDockerCompose(".", workspaceBuilderConfig); err != nil {
		return err
	}
	fmt.Println("Generated workspace docker-compose.yaml")
	return nil
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
