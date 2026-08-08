package p2p

import "khoai-chain/internal/core"

const (
	MsgExecute   = "EXECUTE"    // Client sends command
	MsgNewBlock  = "NEW_BLOCK"  // Case 2: Send if there's a new block
	MsgGetChain  = "GET_CHAIN"  // Case 1: New peer requests data
	MsgSendChain = "SEND_CHAIN" // Case 1: Old peer sends data

	MsgConnectPeer    = "CONNECT_PEER"    // Case 3: New peer connects to old peer
	MsgDisconnectPeer = "DISCONNECT_PEER" // Case 4: Peer disconnects from old peer
	MsgJoinNetwork    = "JOIN_NETWORK"    // Case 5: New peer joins the network
	MsgListPeers      = "LIST_PEERS"      // Case 6: Old peer lists all peers

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

type ConnectPeerRequest struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type DisconnectPeerRequest struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type JoinNetworkRequest struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type ListPeersRequest struct {
	Type string `json:"type"`
}
