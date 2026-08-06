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
	NodeName   string            `yaml:"node_name"`
	Host       string            `yaml:"host"`
	Port       string            `yaml:"port"`
	DBPath     string            `yaml:"db_path"`
	Peers      []string          `yaml:"peers"`
	Chaincodes []ChaincodeConfig `yaml:"chaincodes"` // Remember to define the ChaincodeConfig struct
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
