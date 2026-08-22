package server

import "khoai-chain/internal/core"

const (
	MsgExecute     = "EXECUTE"    // Client sends command
	MsgNewBlock    = "NEW_BLOCK"  // Case 2: Send if there's a new block
	MsgGetChain    = "GET_CHAIN"  // Case 1: New peer requests data
	MsgSendChain   = "SEND_CHAIN" // Case 1: Old peer sends data
	MsgIDENTITY    = "IDENTITY"
	MsgNonce       = "NONCE"
	MsgTransaction = "TRANSACTION"
)

// Request gửi qua TCP
type CommandMessage struct {
	Type      string   `json:"type"`
	Sender    string   `json:"sender"`
	Contract  string   `json:"contract"`
	Function  string   `json:"function"`
	Args      []string `json:"args"`
	Nonce     string   `json:"nonce"`
	Signature string   `json:"signature"`
}

type GetBlocksRequest struct {
	Type   string   `json:"type"`
	Sender string   `json:"sender,omitempty"`
	Hashes []string `json:"hashes"`
}

type SendBlocksRequest struct {
	Type   string        `json:"type"`
	Blocks []*core.Block `json:"blocks"`
}

type CommandIDENTITY struct {
	Type       string `json:"type"`
	PublicKey  string `json:"public_key"`
	Permission string `json:"permission"`
}

type CommandNonce struct {
	Type   string `json:"type"`
	Sender string `json:"sender"`
}

type CommandNewBlock struct {
	Type      string      `json:"type"`
	Sender    string      `json:"sender"`
	Timestamp int64       `json:"timestamp"`
	Block     *core.Block `json:"block"`
	Signature string      `json:"signature"`
}

type CommandTransaction struct {
	Type        string            `json:"type"`
	Sender      string            `json:"sender"`
	Transaction *core.Transaction `json:"transaction"`
}

// Response returned to Client
type ResponseMessage struct {
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}
