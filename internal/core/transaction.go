package core

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// 1. Phần "Ruột" (Nội dung nghiệp vụ bạn muốn gửi)
type TxPayload struct {
	Contract []byte   // Ví dụ: "khoai-token"
	Function []byte   // Ví dụ: "transfer"
	Args     [][]byte // Ví dụ: ["nguyen-van-b", "100"]
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
func NewTransaction(sender, contract, function []byte, args [][]byte) *Transaction {
	tx := &Transaction{
		// ID và Signature sẽ được tính toán sau khi ký
		Timestamp: time.Now().UnixNano(),
		Sender:    sender,
		Payload: TxPayload{
			Contract: contract,
			Function: function,
			Args:     args,
		},
	}
	tx.ID = tx.Hash()
	return tx
}

func (tx *Transaction) Hash() []byte {
	// Đơn giản hóa: Băm sender + timestamp + contract
	data := fmt.Sprintf("%s%d%s%s", tx.Sender, tx.Timestamp, tx.Payload.Contract, tx.Payload.Function)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}
