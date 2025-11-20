package core

// 1. Phần "Ruột" (Nội dung nghiệp vụ bạn muốn gửi)
type TxPayload struct {
	Contract string   // Ví dụ: "khoai-token"
	Function string   // Ví dụ: "transfer"
	Args     []string // Ví dụ: ["nguyen-van-b", "100"]
}

// 2. Phần "Vỏ" (Gói giao dịch hoàn chỉnh)
type Transaction struct {
	ID        []byte // Hash của giao dịch (TxHash)
	Timestamp int64  // Thời gian tạo
	Sender    []byte // Public Key người gửi (Để verify chữ ký)
	Signature []byte // Chữ ký số (Để chứng minh chính chủ)

	Payload TxPayload // Nhét phần ruột vào đây
}

// 3. Hàm tạo nhanh Transaction (Helper/Constructor)
func NewTransaction(sender []byte, contract, function string, args []string) *Transaction {
	return &Transaction{
		// ID và Signature sẽ được tính toán sau khi ký
		Sender: sender,
		Payload: TxPayload{
			Contract: contract,
			Function: function,
			Args:     args,
		},
	}
}
