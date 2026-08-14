package p2p

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// This method sets up the HTTP routes for the node's control plane.
func (s *Server) registerHTTPEndpoints() {
	s.httpMux.HandleFunc("/peers", s.handlePeers)
	s.httpMux.HandleFunc("/peers/remove", s.handleRemovePeer)
	s.httpMux.HandleFunc("/join", s.handleJoin)
	s.httpMux.HandleFunc("/persist-peer", s.handlePersistPeer)
	s.httpMux.HandleFunc("/leave", s.handleLeave)
	s.httpMux.HandleFunc("/leave-notice", s.handleLeaveNotice)
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"peers": s.GetPeerList()})
}

func (s *Server) handleRemovePeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		http.Error(w, "endpoint is required", http.StatusBadRequest)
		return
	}
	s.RemovePeerByEndpoint(body.Endpoint)
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// handleJoin receives a request for this node to connect to a target peer.
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Source string `json:"source"` // Optional, for compatibility
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Target == "" {
		http.Error(w, "target P2P endpoint is required in JSON body", http.StatusBadRequest)
		return
	}

	targetP2P := body.Target
	sourceP2P := s.Endpoint

	if s.IsLocalEndpoint(targetP2P) {
		http.Error(w, "cannot join self", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received join request: this node (%s) will connect to %s\n", sourceP2P, targetP2P)

	// 1. Connect to the target and persist it.
	go s.ConnectToPeer(targetP2P)
	s.persistPeer(targetP2P)

	// 2. Tell the target to connect back and persist us.
	// This ensures the connection is mutual and survives restarts.
	targetHTTP := controlAPIFor(targetP2P)
	err := postJSON(targetHTTP, "/persist-peer", map[string]string{"endpoint": sourceP2P}, nil)
	if err != nil {
		fmt.Printf("Warning: could not ask target %s to persist us back: %v\n", targetP2P, err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"detail": fmt.Sprintf("Join process initiated from %s to %s", sourceP2P, targetP2P),
	})
}

func (s *Server) handleLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for _, peer := range s.GetPeerList() {
		_ = postJSON(controlAPIFor(peer), "/leave-notice", map[string]string{"endpoint": s.Endpoint}, nil)
		s.RemovePeerByEndpoint(peer)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

func (s *Server) handleLeaveNotice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var b struct {
		Endpoint string `json:"endpoint"`
	}
	if json.NewDecoder(r.Body).Decode(&b) != nil || b.Endpoint == "" {
		http.Error(w, "invalid leave notice", http.StatusBadRequest)
		return
	}
	s.RemovePeerByEndpoint(b.Endpoint)
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// handlePersistPeer receives a request from another peer to be added to the persistent peer list.
func (s *Server) handlePersistPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		http.Error(w, "endpoint is required", http.StatusBadRequest)
		return
	}

	peerToPersist := body.Endpoint
	fmt.Printf("Received request to persist peer: %s\n", peerToPersist)

	// Also connect back if not already connected
	if !s.HasPeer(peerToPersist) {
		go s.ConnectToPeer(peerToPersist)
	}
	s.persistPeer(peerToPersist)

	writeJSON(w, http.StatusOK, map[string]string{"status": "persisted"})
}

func controlAPIFor(endpoint string) string {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint
	}
	if _, err := strconv.Atoi(port); err != nil {
		return endpoint
	}
	return net.JoinHostPort(host, "9000")
}

func postJSON(endpoint, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := http.Post("http://"+endpoint+path, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s", res.Status)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
