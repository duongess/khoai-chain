package core

import (
	"crypto/sha256"
	"fmt"
)

// 1. The "Payload" (The business content you want to send)
type TxPayload struct {
	Type     []byte   `json:"type"`
	Sender   []byte   `json:"sender"`
	Contract []byte   `json:"contract"`
	Function []byte   `json:"function"`
	Args     [][]byte `json:"args"`
	Nonce    []byte   `json:"nonce"`
}

// 2. The "Wrapper" (The complete transaction package)
type Transaction struct {
	ID        []byte // Hash of the transaction (TxHash)
	Timestamp int64  // Creation time
	Sender    []byte // Sender's Public Key (To verify the signature)
	Signature []byte // Digital signature (To prove ownership)

	Payload TxPayload // Put the payload here
}

// 3. Quick Transaction creation function (Helper/Constructor)
func NewTransaction(sender, contract, function []byte, args [][]byte, timestamp int64) *Transaction {
	tx := &Transaction{
		Timestamp: timestamp,
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
	// Simplified: Hash sender + timestamp + contract
	data := fmt.Sprintf("%s%d%s%s", tx.Sender, tx.Timestamp, tx.Payload.Contract, tx.Payload.Function)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}
