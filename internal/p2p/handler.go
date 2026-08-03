package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core" // Import để dùng struct Block nếu cần
)

// This function returns ([]byte, error).
// []byte is the JSON data ALREADY PACKAGED to send back to the other party.
func HandleMessage(payload []byte, manager *contract.ContractManager) ([]byte, error) {
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
	}

	errResp := ResponseMessage{Status: "Error", Error: "Unknown command"}
	return json.Marshal(errResp)
}
