package config

import "path/filepath"

const (
	ConfigFileName = "khoai.yaml"
	BuildDir       = "build"
	NodesBaseDir   = "/nodes"
)

func GetNodesBaseDir(BuildDir string) string {
	return filepath.Join(BuildDir, NodesBaseDir)
}
