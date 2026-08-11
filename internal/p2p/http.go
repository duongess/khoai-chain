package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type joinSubmission struct {
	Endpoint    string `json:"endpoint"`
	Bootstrap   string `json:"bootstrap"`
	APIEndpoint string `json:"api_endpoint"`
}

type acceptSubmission struct {
	RequestID string `json:"request_id"`
	NodeID    string `json:"node_id"`
	Endpoint  string `json:"endpoint"`
}

func (s *Server) StartPeerAPI() {
	endpoint, err := s.peerAPIEndpoint()
	if err != nil {
		fmt.Printf("Could not start peer HTTP API: %v\n", err)
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/peers", s.handlePeers)
	mux.HandleFunc("/join", s.handleJoin)
	mux.HandleFunc("/join-requests", s.handleJoinRequests)
	mux.HandleFunc("/join-requests/", s.handleJoinRequestAction)
	mux.HandleFunc("/leave", s.handleLeave)
	mux.HandleFunc("/leave-notice", s.handleLeaveNotice)
	fmt.Printf("Peer management HTTP API listening on %s\n", endpoint)
	if err := http.ListenAndServe(endpoint, mux); err != nil {
		fmt.Printf("Peer HTTP API stopped: %v\n", err)
	}
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"peers": s.GetPeerList()})
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body joinSubmission
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" || body.Bootstrap == "" || body.APIEndpoint == "" {
		http.Error(w, "endpoint, bootstrap and api_endpoint are required", http.StatusBadRequest)
		return
	}
	id, err := requestID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req := &JoinRequest{RequestID: id, NodeID: s.NodeID, Endpoint: body.Endpoint, APIEndpoint: body.APIEndpoint, ExpiresAt: time.Now().Add(joinRequestTTL), outbound: true}
	s.addPendingRequest(req)
	if err := postJSON(peerAPIFor(body.Bootstrap), "/join-requests", req, nil); err != nil {
		s.deletePending(id)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusAccepted, req)
}

func (s *Server) handleJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RequestID == "" || req.Endpoint == "" || req.APIEndpoint == "" {
		http.Error(w, "invalid join request", http.StatusBadRequest)
		return
	}
	req.ExpiresAt = time.Now().Add(joinRequestTTL)
	req.outbound = false
	s.addPendingRequest(&req)
	fmt.Printf("Pending HTTP join request %s from %s; expires at %s\n", req.RequestID, req.Endpoint, req.ExpiresAt.Format(time.RFC3339))
	writeJSON(w, http.StatusAccepted, req)
}

func (s *Server) handleJoinRequestAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/approve") && !strings.HasSuffix(r.URL.Path, "/accept") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	id := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[1]
	if strings.HasSuffix(r.URL.Path, "/approve") {
		if err := s.approveHTTPJoin(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
		return
	}
	var accepted acceptSubmission
	if err := json.NewDecoder(r.Body).Decode(&accepted); err != nil || accepted.RequestID != id {
		http.Error(w, "invalid acceptance", http.StatusBadRequest)
		return
	}
	if !s.acceptHTTPJoin(accepted) {
		http.Error(w, "pending join not found or expired", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) handleLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	for _, peer := range s.GetPeerList() {
		_ = postJSON(peerAPIFor(peer), "/leave-notice", map[string]string{"endpoint": s.Endpoint}, nil)
		s.RemovePeerByEndpoint(peer)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "left"})
}
func (s *Server) handleLeaveNotice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var b struct {
		Endpoint string `json:"endpoint"`
	}
	if json.NewDecoder(r.Body).Decode(&b) != nil || b.Endpoint == "" {
		http.Error(w, "invalid leave notice", 400)
		return
	}
	s.RemovePeerByEndpoint(b.Endpoint)
	writeJSON(w, 200, map[string]string{"status": "removed"})
}

func (s *Server) approveHTTPJoin(id string) error {
	req, ok := s.takePending(id, false)
	if !ok {
		return fmt.Errorf("pending join request %q not found or expired", id)
	}
	accepted := acceptSubmission{RequestID: id, NodeID: s.NodeID, Endpoint: s.Endpoint}
	if err := postJSON(req.APIEndpoint, "/join-requests/"+id+"/accept", accepted, nil); err != nil {
		s.addPendingRequest(req)
		return err
	}
	s.persistPeer(req.Endpoint)
	go s.ConnectToPeer(req.Endpoint)
	return nil
}
func (s *Server) acceptHTTPJoin(a acceptSubmission) bool {
	_, ok := s.takePending(a.RequestID, true)
	if !ok {
		return false
	}
	s.persistPeer(a.Endpoint)
	return true
}
func (s *Server) takePending(id string, outbound bool) (*JoinRequest, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	req, ok := s.PendingRequests[id]
	if !ok || req.outbound != outbound || time.Now().After(req.ExpiresAt) {
		return nil, false
	}
	delete(s.PendingRequests, id)
	return req, true
}
func (s *Server) deletePending(id string) {
	s.lock.Lock()
	delete(s.PendingRequests, id)
	s.lock.Unlock()
}
func (s *Server) peerAPIEndpoint() (string, error) {
	s.lock.RLock()
	configured := s.PeerAPIEndpoint
	endpoint := s.Endpoint
	s.lock.RUnlock()
	if configured != "" {
		return configured, nil
	}
	return peerAPIFor(endpoint), nil
}
func peerAPIFor(endpoint string) string {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return endpoint
	}
	return net.JoinHostPort(host, strconv.Itoa(n+1))
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
