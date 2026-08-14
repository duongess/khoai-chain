package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerComposeUsesSharedP2PNetworkWithoutPublishingTCP(t *testing.T) {
	cfg := &BuilderConfig{
		Network: &NetworkConfig{Name: "khoai-network"},
		Docker:  &DockerConfig{Registry: "local", ImageTag: "test"},
		Organizations: []OrganizationConfig{{
			DisplayName: "Vingroup",
			Nodes: []RuntimeNodeConfig{
				{ID: "hn", Endpoint: "localhost:18081"},
				{ID: "hcm", Endpoint: "localhost:18082"},
				{ID: "dn", Endpoint: "localhost:18083"},
			},
		}},
	}
	dir := t.TempDir()
	if err := GenerateDockerCompose(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	compose := string(data)
	for _, node := range []string{"vingroup-hn", "vingroup-hcm", "vingroup-dn"} {
		if !strings.Contains(compose, node+":") {
			t.Fatalf("missing service %s", node)
		}
	}
	if strings.Count(compose, "- \"9000\"") != 3 {
		t.Fatalf("each node must expose TCP 9000 internally:\n%s", compose)
	}
	// With the new logic, the HTTP control plane is always mapped to port 9000 on the host.
	if strings.Count(compose, "- \"9000:9000\"") != 3 {
		t.Fatalf("each node must publish its HTTP control plane on port 9000:\n%s", compose)
	}
}
