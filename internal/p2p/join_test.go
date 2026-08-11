package p2p

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"khoai-chain/internal/config"
)

func TestJoinRemainsPendingUntilApprovedAndThenPersists(t *testing.T) {
	bConfigPath := filepath.Join(t.TempDir(), "b-config.yaml")
	bConfig := &config.ConfigContent{NodeID: "B", Endpoint: "127.0.0.1:9000"}
	if err := config.SaveConfig(bConfigPath, bConfig); err != nil {
		t.Fatal(err)
	}
	b := NewServer(bConfig.Endpoint, nil)
	b.ConfigurePersistence(bConfig, bConfigPath)

	bConn, aConn := net.Pipe()
	defer aConn.Close()
	go func() {
		NewPeer(bConn, nil).ReadLoop(b)
	}()

	join, err := json.Marshal(JoinNetworkRequest{Type: MsgJoinNetwork, RequestID: "request-1", NodeID: "A", Address: "127.0.0.1:8000"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := aConn.Write(append(join, '\n')); err != nil {
		t.Fatal(err)
	}
	requestID := waitForPendingRequest(t, b)
	if len(b.GetPeerList()) != 0 {
		t.Fatal("a pending join must not become an active peer")
	}

	approveDone := make(chan error, 1)
	go func() { approveDone <- b.ApproveJoin(requestID) }()
	reader := bufio.NewReader(aConn)
	for i := 0; i < 2; i++ {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		var message genericMessage
		if err := json.Unmarshal(line, &message); err != nil {
			t.Fatal(err)
		}
		if i == 0 && message.Type != MsgAcceptJoin {
			t.Fatalf("expected ACCEPT_JOIN, got %s", message.Type)
		}
	}
	if err := <-approveDone; err != nil {
		t.Fatal(err)
	}
	waitForPeer(t, b, "127.0.0.1:8000")

	persistedB, err := config.LoadConfig(bConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persistedB.Peers) != 1 || persistedB.Peers[0] != "127.0.0.1:8000" {
		t.Fatalf("B did not persist A: %#v", persistedB.Peers)
	}
}

func waitForPendingRequest(t *testing.T, server *Server) string {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		server.lock.RLock()
		for id := range server.PendingRequests {
			server.lock.RUnlock()
			return id
		}
		server.lock.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("join request was not pending")
	return ""
}

func waitForPeer(t *testing.T, server *Server, endpoint string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if server.HasPeer(endpoint) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("peer %s was not registered", endpoint)
}

func TestAcceptJoinRequiresMatchingPendingRequest(t *testing.T) {
	server := NewServer("127.0.0.1:1", nil)
	payload, err := json.Marshal(AcceptJoinMessage{Type: MsgAcceptJoin, RequestID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := HandleMessage(payload, server, nil)
	if err != nil {
		t.Fatal(err)
	}
	var result ResponseMessage
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "Error" {
		t.Fatalf("expected missing request to be rejected, got %#v", result)
	}
}
