package config

import "path/filepath"

const (
	ConfigFileName = "khoai.yaml"
	BuildDir       = "build"
	DistDir        = "dist"
	NodesBaseDir   = "/nodes"
	OrgsDir        = "organizations"
)

func GetNodesBaseDir(BuildDir string) string {
	return filepath.Join(BuildDir, NodesBaseDir)
}
