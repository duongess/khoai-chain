package server

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core" // Import để dùng struct Block nếu cần
	"time"
)

// BuildMessageBytes prepares the payload for signing and verification by omitting the signature field.
func BuildMessageBytes(msg CommandMessage) []byte {
	temp := msg
	temp.Signature = ""
	data, _ := json.Marshal(temp)
	return data
}

func HandleTransactionByte(transaction core.TxPayload) []byte {
	data, _ := json.Marshal(transaction)
	return data
}

func HandleBlocknByte(msg CommandNewBlock) []byte {
	temp := msg
	temp.Signature = ""
	data, _ := json.Marshal(temp)
	return data
}

func HandleTransactionSend(transaction *core.Transaction) CommandTransaction {
	return CommandTransaction{
		Type:        MsgTransaction,
		Sender:      string(transaction.Sender),
		Transaction: transaction,
	}
}

func HandleBlockSend(newBlock *core.Block) CommandNewBlock {

	var blockMsg = CommandNewBlock{
		Type:      MsgNewBlock,
		Sender:    hex.EncodeToString(core.PublicKeyNode),
		Timestamp: time.Now().UnixNano(),
		Block:     newBlock,
	}

	temp := blockMsg
	temp.Signature = ""
	data, err := json.Marshal(temp)
	if err != nil {
		return CommandNewBlock{}
	}

	sig := ed25519.Sign(core.PrivateKeyNode, data)
	blockMsg.Signature = hex.EncodeToString(sig)

	return blockMsg
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

		err := core.NM.VerifyAndConsume(msg.Sender, msg.Nonce)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		pubKeyBytes, err := hex.DecodeString(msg.Sender)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key format"})
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key size"})
		}

		sigBytes, err := hex.DecodeString(msg.Signature)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid signature format"})
		}

		messageBytes := BuildMessageBytes(msg)

		err = core.VerifySignature(pubKeyBytes, messageBytes, sigBytes)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		// Call Contract
		result, tx, errContract, err := manager.Execute(
			msg.Sender,
			msg.Contract,
			msg.Function,
			msg.Args,
		)

		var resp ResponseMessage
		if err != nil {
			resp = ResponseMessage{Status: "Error", Error: err.Error()}
		} else if errContract != nil {
			resp = ResponseMessage{Status: "Error", Error: errContract.Error()}
		} else {
			resp = ResponseMessage{Status: "Success", Result: string(result)}
			tx.Signature = msg.Signature
			s.BroadcastNewTransaction(tx)
		}

		return json.Marshal(resp)

	case MsgSendChain:
		var resp SendBlocksRequest
		if err := json.Unmarshal(payload, &resp); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}

		fmt.Printf("Received %d blocks for synchronization...\n", len(resp.Blocks))
		for _, block := range resp.Blocks {
			manager.Chain.AddBlock(block)
		}
		return nil, nil

	case MsgGetChain:
		var req GetBlocksRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}
		if peer != nil && !peer.registered && req.Sender != "" && req.Sender != s.Endpoint {
			peer.Endpoint = req.Sender
			s.registerPeer(peer)
		}

		var commonHash string // Find common point
		for _, hash := range req.Hashes {
			if manager.Chain.DB.HasKey([]byte(hash)) {
				commonHash = hash
				break
			}
		}
		var blocksToSend []*core.Block
		if commonHash == "" {

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
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}
		core.PublicKeyPeers[msg.PublicKey] = msg.Permission

		resp := ResponseMessage{Status: "Success"}
		return json.Marshal(resp)

	case MsgNonce:
		var msg CommandNonce
		if err := json.Unmarshal(payload, &msg); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}

		var nonce, err = core.NM.GenerateChallenge(msg.Sender)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		return json.Marshal(ResponseMessage{Status: "Success", Result: nonce})

	case MsgTransaction:
		var msg CommandTransaction
		if err := json.Unmarshal(payload, &msg); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}
		pubKeyBytes, err := hex.DecodeString(msg.Sender)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key format"})
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key size"})
		}

		sigBytes, err := hex.DecodeString(string(msg.Transaction.Signature))
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid signature format"})
		}

		messageBytes := HandleTransactionByte(msg.Transaction.Payload)

		err = core.VerifySignature(pubKeyBytes, messageBytes, sigBytes)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		_, err = manager.AddAndCheckMine(msg.Transaction)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		return json.Marshal(ResponseMessage{Status: "Success"})

	case MsgNewBlock:
		var msg CommandNewBlock
		if err := json.Unmarshal(payload, &msg); err != nil {
			resp := ResponseMessage{Status: "Error", Error: "Invalid JSON for EXECUTE"}
			return json.Marshal(resp)
		}
		pubKeyBytes, err := hex.DecodeString(msg.Sender)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key format"})
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid public key size"})
		}

		sigBytes, err := hex.DecodeString(msg.Signature)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid signature format"})
		}

		var messageBytes = HandleBlocknByte(msg)
		err = core.VerifySignature(pubKeyBytes, messageBytes, sigBytes)
		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		for _, transaction := range msg.Block.Transactions {
			pubKeyBytes, err := hex.DecodeString(transaction.Sender)
			if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
				return json.Marshal(ResponseMessage{Status: "Error", Error: "invalid tx sender public key"})
			}

			var messageBytes = HandleTransactionByte(transaction.Payload)
			err = core.VerifySignature(ed25519.PublicKey(transaction.Payload.Sender), messageBytes, []byte(transaction.Signature))
			if err != nil {
				return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
			}

			_, _, err = manager.ExecuteOnly(
				transaction.Sender,
				transaction.Payload.Contract,
				transaction.Payload.Function,
				transaction.Payload.Args,
			)
			if err != nil {
				return json.Marshal(ResponseMessage{
					Status: "Error",
					Error:  fmt.Sprintf("invalid transaction in block: %s", err.Error()),
				})
			}
		}

		err = manager.Commit()

		if err != nil {
			return json.Marshal(ResponseMessage{Status: "Error", Error: err.Error()})
		}

		manager.Chain.AddBlock(msg.Block)

		return json.Marshal(ResponseMessage{Status: "Success"})
	}

	errResp := ResponseMessage{Status: "Error", Error: "Unknown command"}
	return json.Marshal(errResp)
}
