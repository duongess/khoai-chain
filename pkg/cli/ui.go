package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"khoai-chain/internal/config"
	"khoai-chain/internal/server"
	"net/http"
)

//go:embed frontend/dist
var uiFiles embed.FS

type response struct {
	Status string `json:"status"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

const (
	SUCCESS = "success"
	ERROR   = "error"
)

func RunUI(conf *config.ConfigContent, srv *server.Server) error {
	// 1. Initialize embedded file system for static UI assets
	distFS, err := fs.Sub(uiFiles, "frontend/dist")
	if err != nil {
		return fmt.Errorf("error initializing static filesystem: %w", err)
	}

	mux := http.NewServeMux()

	// 2. Serve static frontend files
	mux.Handle("/docs", http.FileServer(http.FS(distFS)))

	mux.HandleFunc("/api/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{
			Status: SUCCESS,
			Result: map[string]any{
				"name":         conf.NodeName,
				"organization": conf.Organization,
				"permission":   conf.Permission,
			},
		})
	})

	// 3. Config API endpoint
	mux.HandleFunc("/api/contracts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		contractsMeta, err := srv.Contracts.GetMetadataList()
		if err != nil {
			json.NewEncoder(w).Encode(response{
				Status: ERROR,
				Error:  err.Error(),
			})
		} else {
			json.NewEncoder(w).Encode(response{
				Status: SUCCESS,
				Result: contractsMeta,
			})
		}
	})

	mux.HandleFunc("/api/p2p/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(response{
				Status: ERROR,
				Error:  "Method not allowed",
			})
			return
		}

		var rawData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&rawData); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response{
				Status: ERROR,
				Error:  fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		result, err := srv.SendMessageToNode(rawData)

		if err != nil {
			json.NewEncoder(w).Encode(response{
				Status: ERROR,
				Error:  err.Error(),
			})
			return
		}

		// Return standard response structure
		json.NewEncoder(w).Encode(response{
			Status: SUCCESS,
			Result: result,
		})
	})

	fmt.Printf("The UI is currently running at http://localhost%s (Connect to Node: %s)\n", config.Endpoint, conf.P2PEndpoint)

	return http.ListenAndServe(config.Endpoint, mux)
}
