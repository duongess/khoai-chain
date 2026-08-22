package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"time"
)

type Block struct {
	Timestamp     int64
	PrevBlockHash string
	Hash          string
	Transactions  []*Transaction
	Height        int
}

func NewBlock(txs []*Transaction, prevBlockHash string, height int) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  txs,
		PrevBlockHash: prevBlockHash,
		Height:        height,
	}
	// Calculate hash upon creation
	block.Hash = block.CalculateHash()
	return block
}

// NewGenesisBlock creates the first block (Genesis Block)
func NewGenesisBlock() *Block {
	// Create a coinbase transaction
	genesisTx := NewTransaction("System", "KhoaiChain", "Init", []string{"Genesis"}, 0)
	return NewBlock([]*Transaction{genesisTx}, "", 0)
}

func (b *Block) CalculateHash() string {
	timestampStr := fmt.Sprintf("%d", b.Timestamp)

	headers := b.PrevBlockHash + timestampStr

	for _, tx := range b.Transactions {
		headers += tx.ID
	}

	hashBytes := sha256.Sum256([]byte(headers))

	return hex.EncodeToString(hashBytes[:])
}

// Serialize: Converts a Block into a byte slice (for DB storage or network transfer)
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock: Converts a byte slice back into a Block
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}
