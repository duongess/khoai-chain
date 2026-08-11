package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type ConfigContent struct {
	NodeName     string            `yaml:"node_name"`
	DBPath       string            `yaml:"db_path"`
	Chaincodes   []ChaincodeConfig `yaml:"chaincodes"`
	Organization string            `yaml:"organization"`
	NodeID       string            `yaml:"node_id"`
	DisplayName  string            `yaml:"display_name"`
	Endpoint     string            `yaml:"endpoint"`
	Peers        []string          `yaml:"peers,omitempty"`
}

func LoadConfig(filePath string) (*ConfigContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %v", err)
	}

	var conf ConfigContent
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("YAML format error: %v", err)
	}

	return &conf, nil
}

// SaveConfig persists a runtime node configuration. The temporary-file rename
// prevents an interrupted write from leaving config.yaml partially written.
func SaveConfig(filePath string, conf *ConfigContent) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("could not encode config: %w", err)
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("could not write temporary config: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("could not replace config: %w", err)
	}
	return nil
}

// BuildExe builds the executable from the source code in a given directory.
// This simplified version no longer deals with encrypted zips.
func BuildExe(outputDir string) error {
	fmt.Println("Starting build process...")

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("could not create output directory %s: %w", outputDir, err)
	}

	// 1. Define output path for the executable
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("Error getting absolute path: %v", err)
	}

	outputName := "khoai-node"
	if runtime.GOOS == "windows" {
		outputName += ".exe"
	}
	outputExe := filepath.Join(absOutputDir, outputName)

	fmt.Printf("Creating executable at: %s\n", outputExe)

	// 2. Build the Go program
	cmd := exec.Command("go", "build", "-o", outputExe, "./cmd/node")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Build failed: %v", err)
	}

	fmt.Println("Build successful!")
	return nil
}
