package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core" // Import để dùng struct Block nếu cần
	"time"
)

// This function returns ([]byte, error).
// []byte is the JSON data ALREADY PACKAGED to send back to the other party.
func HandleMessage(payload []byte, s *Server, manager *contract.ContractManager) ([]byte, error) {
	return handleMessage(payload, s, manager, nil)
}

// genericMessage là một cấu trúc chung chỉ để lấy ra trường 'type' từ JSON.
type genericMessage struct {
	Type string `json:"type"`
}

const legacyMsgListPeers = "LIST_PEERS"

func handleMessage(payload []byte, s *Server, manager *contract.ContractManager, peer *Peer) ([]byte, error) {
	fmt.Printf("Processing data: %s\n", string(payload))

	var baseMsg genericMessage
	// If JSON is malformed from the start -> Return JSON error to Client
	if err := json.Unmarshal(payload, &baseMsg); err != nil {
		// Package JSON error here
		resp := ResponseMessage{Status: "Error", Error: "Invalid JSON format"}
		return json.Marshal(resp)
	}

	switch baseMsg.Type {

	// --- CASE 1: CLIENT SENDS COMMAND (RETURNS ResponseMessage) ---
	case MsgExecute:
		var msg CommandMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}

		var argsBytes [][]byte
		for _, arg := range msg.Args {
			argsBytes = append(argsBytes, []byte(arg))
		}

		// Gọi Contract
		// Call Contract
		result, err := manager.Execute(
			[]byte(msg.Sender),
			[]byte(msg.Contract),
			[]byte(msg.Function),
			argsBytes,
		)
		// Package ResponseMessage here
		var resp ResponseMessage
		if err != nil {
			resp = ResponseMessage{Status: "Error", Error: err.Error()}
		} else {
			resp = ResponseMessage{Status: "Success", Result: string(result)}
		}
		// Return marshaled bytes
		return json.Marshal(resp)

	case MsgSendChain:
		var resp SendBlocksRequest
		if err := json.Unmarshal(payload, &resp); err != nil {
			return nil, err
		}

		fmt.Printf("Received %d blocks for synchronization...\n", len(resp.Blocks))
		for _, block := range resp.Blocks {
			manager.Chain.AddBlock(block)
		}
		return nil, nil

	case MsgGetChain:
		var req GetBlocksRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, err
		}

		var commonHash []byte // Find common point
		for _, hash := range req.Hashes {
			if manager.Chain.DB.HasKey(hash) {
				commonHash = hash
				break
			}
		}
		var blocksToSend []*core.Block
		if commonHash == nil {

			// No common point found (Full Sync)
			fmt.Println("No common point found (Full Sync)")
			blocksToSend = manager.Chain.GetAllBlock()
		} else {
			fmt.Printf("Common point at: %x\n", commonHash)
			blocksToSend = manager.Chain.GetBlockAfter(commonHash)
		}

		resp := SendBlocksRequest{
			Type:   MsgSendChain,
			Blocks: blocksToSend,
		}
		return json.Marshal(resp)

	case MsgJoinNetwork:
		var req JoinNetworkRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for JOIN_NETWORK"}
			return json.Marshal(resp)
		}

		// CLI asks its local node to begin a JOIN. The node subsequently sends a
		// normal JOIN_NETWORK request directly to the bootstrap peer.
		if req.Bootstrap != "" {
			if !s.IsLocalEndpoint(req.Address) {
				resp := ResponseMessage{Status: "Error", Error: "JOIN_NETWORK control request must target the local node"}
				return json.Marshal(resp)
			}
			go s.JoinNetwork(req.Bootstrap)
			resp := ResponseMessage{Status: "Success", Result: fmt.Sprintf("Join request via %s initiated.", req.Bootstrap)}
			return json.Marshal(resp)
		}
		if req.Address == "" || req.Address == s.Endpoint {
			resp := ResponseMessage{Status: "Error", Error: "Invalid joining node address"}
			return json.Marshal(resp)
		}
		if req.RequestID == "" {
			resp := ResponseMessage{Status: "Error", Error: "JOIN_NETWORK requires request_id"}
			return json.Marshal(resp)
		}
		if peer == nil {
			resp := ResponseMessage{Status: "Error", Error: "JOIN_NETWORK requires a peer connection"}
			return json.Marshal(resp)
		}

		// Do not register or persist the inbound socket here. It remains an
		// in-memory request until an operator sends ACCEPT_JOIN for this exact ID.
		peer.Endpoint = req.Address
		s.addPendingRequest(&JoinRequest{
			RequestID: req.RequestID,
			NodeID:    req.NodeID,
			Endpoint:  req.Address,
			ExpiresAt: time.Now().Add(joinRequestTTL),
			peer:      peer,
		})
		fmt.Printf("Pending join request %s from node %s at %s; expires at %s\n", req.RequestID, req.NodeID, req.Address, time.Now().Add(joinRequestTTL).Format(time.RFC3339))
		return nil, nil

	case MsgAcceptJoin:
		var req AcceptJoinMessage
		if err := json.Unmarshal(payload, &req); err != nil || req.RequestID == "" {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "Invalid JSON for ACCEPT_JOIN"})
		}
		if err := s.ApproveJoin(req.RequestID); err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}
		return json.Marshal(ResponseMessage{Status: "Success", Result: fmt.Sprintf("Join request %s accepted.", req.RequestID)})

	case MsgLeaveNetwork:
		var req LeaveNetworkMessage
		if err := json.Unmarshal(payload, &req); err != nil || req.Address == "" {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for LEAVE_NETWORK"}
			return json.Marshal(resp)
		}
		if req.Address == s.Endpoint {
			go s.LeaveNetwork()
			return json.Marshal(ResponseMessage{Status: "Success", Result: "Leaving network."})
		}
		s.RemovePeerByEndpoint(req.Address)
		return nil, nil

	case MsgPeerList, legacyMsgListPeers:
		var req PeerListMessage
		if err := json.Unmarshal(payload, &req); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for PEER_LIST"}
			return json.Marshal(resp)
		}
		if req.Request {
			return json.Marshal(PeerListMessage{Type: MsgPeerList, Sender: s.Endpoint, Peers: s.GetPeerList()})
		}
		// A regular ConnectToPeer starts with PEER_LIST as a small hello. The
		// accepted side becomes a peer only after that hello, so raw CLI sockets
		// still never enter the peer list.
		if peer != nil && !peer.registered && req.Sender != "" && req.Sender != s.Endpoint {
			peer.Endpoint = req.Sender
			peer.joinPeers = s.GetPeerList()
		}
		for _, address := range req.Peers {
			if address != s.Endpoint && address != req.Sender {
				go s.ConnectToPeer(address)
			}
		}
		return nil, nil

	}

	errResp := ResponseMessage{Status: "Error", Error: "Unknown command"}
	return json.Marshal(errResp)
}
