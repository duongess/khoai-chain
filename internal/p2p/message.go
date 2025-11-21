package p2p

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

// Response trả về cho Client
type ResponseMessage struct {
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}
