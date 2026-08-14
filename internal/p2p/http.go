package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"khoai-chain/internal/config"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// This method sets up the HTTP routes for the node's control plane.
func (s *Server) registerHTTPEndpoints() {
	s.httpMux.HandleFunc("/peers", s.handlePeers)
	s.httpMux.HandleFunc("/peers/remove", s.handleRemovePeer)
	s.httpMux.HandleFunc("/join", s.handleCreateJoinRequest)
	s.httpMux.HandleFunc("/join-requests", s.handleListJoinRequests)
	s.httpMux.HandleFunc("/approve", s.handleApproveJoin)
	s.httpMux.HandleFunc("/add-peer", s.handleAddPeer)
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

// handleCreateJoinRequest receives a request from a potential new peer.
// It creates a pending request that must be approved by an admin.
func (s *Server) handleCreateJoinRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SourceEndpoint string `json:"source_endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SourceEndpoint == "" {
		http.Error(w, "source_endpoint is required in JSON body", http.StatusBadRequest)
		return
	}

	id, err := requestID()
	if err != nil {
		http.Error(w, "could not generate request ID", http.StatusInternalServerError)
		return
	}

	req := &JoinRequest{
		RequestID:      id,
		SourceEndpoint: body.SourceEndpoint,
		ExpiresAt:      time.Now().Add(config.JoinRequestTTL),
	}
	s.addPendingRequest(req)

	fmt.Printf("Received join request from %s. Request ID: %s\n", req.SourceEndpoint, req.RequestID)
	writeJSON(w, http.StatusAccepted, req)
}

func (s *Server) handleListJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	requests := make([]*JoinRequest, 0, len(s.PendingRequests))
	for _, req := range s.PendingRequests {
		requests = append(requests, req)
	}
	writeJSON(w, http.StatusOK, map[string]any{"join_requests": requests})
}

// handleApproveJoin finalizes a connection from a pending request.
func (s *Server) handleApproveJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RequestID == "" {
		http.Error(w, "request_id is required", http.StatusBadRequest)
		return
	}

	req, ok := s.takePendingRequest(body.RequestID)
	if !ok {
		http.Error(w, "join request not found or expired", http.StatusNotFound)
		return
	}

	sourceEndpoint := req.SourceEndpoint
	targetEndpoint := s.Endpoint

	fmt.Printf("Approving join request %s: this node (%s) will connect to %s\n", req.RequestID, targetEndpoint, sourceEndpoint)

	// 1. Connect to the source and persist it.
	go s.ConnectToPeer(sourceEndpoint)
	s.persistPeer(sourceEndpoint)

	// 2. Tell the source to connect back and persist us.
	sourceHTTP := controlAPIFor(sourceEndpoint)
	err := postJSON(sourceHTTP, "/add-peer", map[string]string{"endpoint": targetEndpoint}, nil)
	if err != nil {
		// This is not a fatal error, but should be logged. The connection might be one-way for now.
		fmt.Printf("Warning: could not ask source %s to persist us back: %v\n", sourceEndpoint, err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "approved",
		"detail": fmt.Sprintf("Join process approved for %s", sourceEndpoint),
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

// handleAddPeer receives a request from another peer (during its approval process)
// to be added to the persistent peer list.
func (s *Server) handleAddPeer(w http.ResponseWriter, r *http.Request) {
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

	peerToAdd := body.Endpoint
	fmt.Printf("Received request to add peer: %s\n", peerToAdd)

	// Also connect back if not already connected
	if !s.HasPeer(peerToAdd) {
		go s.ConnectToPeer(peerToAdd)
	}
	s.persistPeer(peerToAdd)

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
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

func requestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
