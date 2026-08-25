package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"khoai-chain/internal/config"
	"net"
	"net/http"
)

//go:embed frontend/dist
var uiFiles embed.FS

type response struct {
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func RunUI(targetNode string, contract interface{}) error {
	// 1. Initialize embedded file system for static UI assets
	distFS, err := fs.Sub(uiFiles, "frontend/dist")
	if err != nil {
		return fmt.Errorf("error initializing static filesystem: %w", err)
	}

	mux := http.NewServeMux()

	// 2. Serve static frontend files
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// 3. Config API endpoint
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"TargetNode": targetNode,
			"contract":   contract,
		})
	})

	// 4. Bridge API: Forward HTTP payload to P2P TCP node
	mux.HandleFunc("/api/p2p/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(response{
				Status: "error",
				Error:  "Method not allowed",
			})
			return
		}

		var rawData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&rawData); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response{
				Status: "error",
				Error:  fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		// Marshal the payload to send over raw TCP socket
		payloadBytes, err := json.Marshal(rawData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to marshal payload: %v", err),
			})
			return
		}

		// Dial TCP connection to the target P2P node
		conn, err := net.Dial("tcp", targetNode)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to connect to target P2P node %s: %v", targetNode, err),
			})
			return
		}
		defer conn.Close()

		// Send payload followed by a newline (line-based protocol)
		if _, err := conn.Write(append(payloadBytes, '\n')); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to write data to TCP socket: %v", err),
			})
			return
		}

		// Optionally read response from the TCP node if expected
		// (Assuming a simple acknowledgement or response string)
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		var tcpResult string
		if err == nil && n > 0 {
			tcpResult = string(buffer[:n])
		} else {
			tcpResult = "Packet transmitted successfully via TCP bridge"
		}

		// Return standard response structure
		json.NewEncoder(w).Encode(response{
			Status: "success",
			Result: tcpResult,
		})
	})

	fmt.Printf("The UI is currently running at http://localhost%s (Connect to Node: %s)\n", config.Endpoint, targetNode)

	return http.ListenAndServe(config.Endpoint, mux)
}
