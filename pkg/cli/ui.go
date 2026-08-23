package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"khoai-chain/internal/config"
	"net/http"
)

//go:embed frontend/dist
var uiFiles embed.FS

func RunUI(targetNode string) error {
	// 1. Chi parse file HTML de lam template truyen bien
	distFS, err := fs.Sub(uiFiles, "frontend/dist")
	if err != nil {
		return fmt.Errorf("error parse template: %w", err)
	}

	mux := http.NewServeMux()
	// 4. Dinh tuyen cho HTML va tra ve loi 404 neu go sai duong dan
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"TargetNode": targetNode,
		})
	})

	fmt.Printf("The UI is currently running at http://localhost%s (Connect to Node: %s)\n", config.Endpoint, targetNode)

	return http.ListenAndServe(config.Endpoint, mux)
}
