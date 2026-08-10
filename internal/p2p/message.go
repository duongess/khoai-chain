package p2p

import "khoai-chain/internal/core"

const (
	MsgExecute   = "EXECUTE"    // Client sends command
	MsgNewBlock  = "NEW_BLOCK"  // Case 2: Send if there's a new block
	MsgGetChain  = "GET_CHAIN"  // Case 1: New peer requests data
	MsgSendChain = "SEND_CHAIN" // Case 1: Old peer sends data

	// Dynamic peer membership messages. Server peer operations remain local
	// operations; they are deliberately not exposed as protocol messages.
	MsgJoinNetwork  = "JOIN_NETWORK"
	MsgAcceptJoin   = "ACCEPT_JOIN"
	MsgLeaveNetwork = "LEAVE_NETWORK"
	MsgPeerList     = "PEER_LIST"
)

// Request gửi qua TCP
type CommandMessage struct {
	Type     string   `json:"type"`
	Sender   string   `json:"sender"`
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

// Response returned to Client
type ResponseMessage struct {
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type JoinNetworkRequest struct {
	Type      string `json:"type"`
	Address   string `json:"address"`             // Joining node's listening endpoint.
	Bootstrap string `json:"bootstrap,omitempty"` // Set only by the local CLI command.
}

type AcceptJoinMessage struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type LeaveNetworkMessage struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type PeerListMessage struct {
	Type    string   `json:"type"`
	Sender  string   `json:"sender,omitempty"`
	Peers   []string `json:"peers,omitempty"`
	Request bool     `json:"request,omitempty"`
}
