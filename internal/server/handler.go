package server

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

// genericMessage là một cấu trúc chung chỉ để lấy ra trường 'type' từ JSON.
type genericMessage struct {
	Type string `json:"type"`
}

func handleMessage(payload []byte, s *Server, manager *contract.ContractManager, peer *Peer) ([]byte, error) {
	fmt.Printf("Processing data: %s\n", string(payload))

	var baseMsg genericMessage
	if err := json.Unmarshal(payload, &baseMsg); err != nil {
		resp := ResponseMessage{Status: "Error", Error: "Invalid JSON format"}
		return json.Marshal(resp)
	}

	// If the 'type' field is empty, it's likely a response to a request we sent earlier.
	// Response messages (like ResponseMessage) don't have a 'type'.
	// We should not process them as new requests, to avoid loops.
	if baseMsg.Type == "" {
		var respMsg ResponseMessage
		// We try to unmarshal it as a ResponseMessage to confirm.
		if err := json.Unmarshal(payload, &respMsg); err == nil && (respMsg.Status == "Success" || respMsg.Status == "Error") {
			// It is indeed a response. Log it and do nothing else.
			fmt.Printf("Received a response from peer: Status=%s, Result='%s', Error='%s'\n", respMsg.Status, respMsg.Result, respMsg.Error)
			return nil, nil // Return nil to send no further messages.
		}
		// If it's not a ResponseMessage and has no type, it's an unknown command.
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

		var err = core.NM.VerifyAndConsume(msg.Nonce, msg.Sender)
		if err != nil {
			json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

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
		if peer != nil && !peer.registered && req.Sender != "" && req.Sender != s.Endpoint {
			peer.Endpoint = req.Sender
			s.registerPeer(peer)
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

	case MsgIDENTITY:
		var msg CommandIDENTITY
		if err := json.Unmarshal(payload, &msg); err != nil {
			return nil, err
		}
		core.PublicKeyPeers[msg.PublicKey] = msg.Permission

		resp := ResponseMessage{Status: "Success"}
		return json.Marshal(resp)

	case MsgNonce:
		var msg CommandNonce
		if err := json.Unmarshal(payload, &msg); err != nil {
			return nil, err
		}

		core.NM.GenerateChallenge(msg.Sender)
	}

	errResp := ResponseMessage{Status: "Error", Error: "Unknown command"}
	return json.Marshal(errResp)
}
