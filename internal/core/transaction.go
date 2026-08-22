package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// 1. The "Payload" (The business content you want to send)
type TxPayload struct {
	Type      string   `json:"type"`
	Sender    string   `json:"sender"`
	Contract  string   `json:"contract"`
	Function  string   `json:"function"`
	Args      []string `json:"args"`
	Nonce     string   `json:"nonce"`
	Signature string   `json:"signature"`
}

// 2. The "Wrapper" (The complete transaction package)
type Transaction struct {
	ID        string // Hash of the transaction (TxHash)
	Timestamp int64  // Creation time

	Payload TxPayload // Put the payload here
}

// 3. Quick Transaction creation function (Helper/Constructor)
func NewTransaction(payload TxPayload, timestamp int64) *Transaction {
	tx := &Transaction{
		Timestamp: timestamp,
		Payload:   payload,
	}
	tx.ID = tx.Hash()
	return tx
}

func (tx *Transaction) Hash() string {
	data := fmt.Sprintf("%s%d%s%s", tx.Payload.Sender, tx.Timestamp, tx.Payload.Contract, tx.Payload.Function)
	hashBytes := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hashBytes[:])
}
