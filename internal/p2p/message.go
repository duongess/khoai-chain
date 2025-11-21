package p2p

import "khoai-chain/internal/core"

const (
	MsgExecute   = "EXECUTE"    // Client gửi lệnh
	MsgNewBlock  = "NEW_BLOCK"  // Case 2: Có block mới thì gửi
	MsgGetChain  = "GET_CHAIN"  // Case 1: Peer mới xin dữ liệu
	MsgSendChain = "SEND_CHAIN" // Case 1: Peer cũ trả dữ liệu
)

// Request gửi qua TCP
type CommandMessage struct {
	Type     string   `json:"type"`
	Contract string   `json:"contract"`
	Function string   `json:"function"`
	Args     []string `json:"args"`
}

type GetBlocksRequest struct {
	Type   string   `json:"type"`
	Hashes [][]byte `json:"hashes"`
}

type SendBlocksRequest struct {
	Type   string        `json:"type"`
	Blocks []*core.Block `json:"blocks"`
}

// Response trả về cho Client
type ResponseMessage struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}
