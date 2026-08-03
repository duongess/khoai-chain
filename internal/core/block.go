package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"time"
)

type Block struct {
	Timestamp     int64
	PrevBlockHash []byte
	Hash          []byte
	Transactions  []*Transaction
	Height        int
}

func NewBlock(txs []*Transaction, prevBlockHash []byte, height int) *Block {
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
	genesisTx := NewTransaction([]byte("System"), []byte("KhoaiChain"), []byte("Init"), [][]byte{[]byte("Genesis")})
	return NewBlock([]*Transaction{genesisTx}, []byte{}, 0)
}

func (b *Block) CalculateHash() []byte {
	// Concatenate all data to be hashed
	// (Simplified, a Merkle Tree would be used in practice)
	timestamp := []byte(string(rune(b.Timestamp)))
	headers := bytes.Join([][]byte{b.PrevBlockHash, timestamp}, []byte{})

	// Add the hash of the transactions
	for _, tx := range b.Transactions {
		headers = bytes.Join([][]byte{headers, tx.ID}, []byte{})
	}

	hash := sha256.Sum256(headers)
	return hash[:]
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
