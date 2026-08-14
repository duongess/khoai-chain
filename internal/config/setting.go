package config

import "path/filepath"

const (
	ConfigFileName = "khoai-config.yaml"
	BuildDir       = "build"
	DistDir        = "dist"
	NodesBaseDir   = "/nodes"
	OrgsDir        = "organizations"

	JoinRequestTTL = 5 * 60 // 5 minutes in seconds
)

func GetNodesBaseDir(BuildDir string) string {
	return filepath.Join(BuildDir, NodesBaseDir)
}
