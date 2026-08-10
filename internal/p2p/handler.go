package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core" // Import để dùng struct Block nếu cần
)

// This function returns ([]byte, error).
// []byte is the JSON data ALREADY PACKAGED to send back to the other party.
func HandleMessage(payload []byte, s *Server, manager *contract.ContractManager) ([]byte, error) {
	return handleMessage(payload, s, manager, nil)
}

func handleMessage(payload []byte, s *Server, manager *contract.ContractManager, peer *Peer) ([]byte, error) {
	fmt.Printf("Processing data: %s\n", string(payload))

	var msg CommandMessage
	// If JSON is malformed from the start -> Return JSON error to Client
	if err := json.Unmarshal(payload, &msg); err != nil {
		// Package JSON error here
		resp := ResponseMessage{Status: "Error", Error: "Invalid JSON format"}
		return json.Marshal(resp)
	}

	switch msg.Type {

	// --- CASE 1: CLIENT SENDS COMMAND (RETURNS ResponseMessage) ---
	case MsgExecute:
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
			if req.Address != s.Endpoint {
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
		if peer == nil {
			resp := ResponseMessage{Status: "Error", Error: "JOIN_NETWORK requires a peer connection"}
			return json.Marshal(resp)
		}

		// Do not register the inbound socket here. ReadLoop promotes it only
		// after ACCEPT_JOIN and PEER_LIST have been written successfully.
		peer.Endpoint = req.Address
		peer.joinPeers = s.GetPeerList()
		return json.Marshal(AcceptJoinMessage{Type: MsgAcceptJoin, Address: s.Endpoint})

	case MsgAcceptJoin:
		// ACCEPT_JOIN is consumed synchronously by Server.JoinNetwork before the
		// socket is promoted to a peer, so a received duplicate is harmless.
		return nil, nil

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

	case MsgPeerList:
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
