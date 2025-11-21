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
	// Tính hash ngay khi tạo
	block.Hash = block.CalculateHash()
	return block
}

// NewGenesisBlock tạo khối đầu tiên (Khối tổ)
func NewGenesisBlock() *Block {
	// Tạo một giao dịch mồi
	genesisTx := NewTransaction([]byte("System"), []byte("KhoaiChain"), []byte("Init"), [][]byte{[]byte("Genesis")})
	return NewBlock([]*Transaction{genesisTx}, []byte{}, 0)
}

func (b *Block) CalculateHash() []byte {
	// Nối tất cả dữ liệu lại để băm
	// (Làm đơn giản, thực tế dùng Merkle Tree)
	timestamp := []byte(string(rune(b.Timestamp)))
	headers := bytes.Join([][]byte{b.PrevBlockHash, timestamp}, []byte{})

	// Cộng thêm hash của các giao dịch
	for _, tx := range b.Transactions {
		headers = bytes.Join([][]byte{headers, tx.ID}, []byte{})
	}

	hash := sha256.Sum256(headers)
	return hash[:]
}

// Serialize: Biến Block thành mảng byte (để lưu xuống DB hoặc gửi qua mạng)
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock: Biến mảng byte ngược lại thành Block
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}
