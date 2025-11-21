package p2p

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
