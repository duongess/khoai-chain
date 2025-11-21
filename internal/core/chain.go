package core

import (
	"fmt"
	"khoai-chain/internal/database"
	"os"
)

const (
	lastHashKey = "lh" // Key đặc biệt để lưu Hash của block mới nhất
)

type Blockchain struct {
	LastHash []byte            // Hash của khối mới nhất (Cái đuôi)
	DB       *database.Storage // Kết nối đến Database
}

// InitBlockchain: Khởi tạo chuỗi.
// Nếu DB chưa có gì -> Tạo Genesis.
// Nếu DB có rồi -> Load LastHash lên.
func InitBlockchain(db *database.Storage) *Blockchain {
	var lastHash []byte

	// Kiểm tra xem trong DB đã có Blockchain chưa
	if !db.HasKey([]byte(lastHashKey)) {
		fmt.Println("🌟 Chưa có Blockchain, đang tạo Genesis Block...")

		// 1. Tạo Genesis Block
		genesis := NewGenesisBlock()

		// 2. Lưu Block vào DB (Key=Hash, Value=Serialize)
		err := db.SetData(genesis.Hash, genesis.Serialize())
		if err != nil {
			fmt.Println("❌ Lỗi lưu Genesis:", err)
			os.Exit(1)
		}

		// 3. Lưu con trỏ LastHash
		err = db.SetData([]byte(lastHashKey), genesis.Hash)
		if err != nil {
			fmt.Println("❌ Lỗi lưu LastHash:", err)
			os.Exit(1)
		}

		lastHash = genesis.Hash
	} else {
		// Nếu có rồi thì lấy Hash cuối cùng ra
		var err error
		lastHash, err = db.GetData([]byte(lastHashKey))
		if err != nil {
			fmt.Println("❌ Lỗi đọc LastHash:", err)
			os.Exit(1)
		}
		fmt.Printf("🔄 Đã load Blockchain. LastHash: %x\n", lastHash)
	}

	return &Blockchain{LastHash: lastHash, DB: db}
}

// MineBlock: Đóng gói giao dịch thành khối mới và thêm vào chuỗi
func (bc *Blockchain) MineBlock(txs []*Transaction) *Block {
	// 1. Lấy block cuối cùng để biết Hash và Height của nó
	lastBlock, err := bc.GetBlock(bc.LastHash)
	if err != nil {
		fmt.Println("❌ Không tìm thấy block cuối:", err)
		return nil
	}

	// 2. Tạo Block mới (Height tăng lên 1)
	newBlock := NewBlock(txs, bc.LastHash, lastBlock.Height+1)

	// 3. Lưu Block mới vào DB
	err = bc.DB.SetData(newBlock.Hash, newBlock.Serialize())
	if err != nil {
		fmt.Println("❌ Lỗi lưu block mới:", err)
		return nil
	}

	// 4. Cập nhật LastHash
	err = bc.DB.SetData([]byte(lastHashKey), newBlock.Hash)
	if err != nil {
		fmt.Println("❌ Lỗi cập nhật LastHash:", err)
		return nil
	}

	// 5. Cập nhật struct trong bộ nhớ
	bc.LastHash = newBlock.Hash

	fmt.Printf("⛏️  Đã đào được Block #%d [%x]\n", newBlock.Height, newBlock.Hash)
	return newBlock
}

// AddBlock: Hàm này dùng cho P2P Sync (Khi nhận Block từ thằng khác)
// Khác với MineBlock là Block này đã có sẵn Hash và dữ liệu, chỉ cần lưu thôi
func (bc *Blockchain) AddBlock(block *Block) {
	// TODO: Cần thêm logic kiểm tra hợp lệ (Validate) ở đây nữa

	err := bc.DB.SetData(block.Hash, block.Serialize())
	if err != nil {
		fmt.Println("❌ Lỗi lưu block từ peer:", err)
		return
	}

	// Kiểm tra xem block này có cao hơn block hiện tại của mình không
	// Nếu cao hơn thì cập nhật LastHash (Longest Chain Rule)
	lastBlock, _ := bc.GetBlock(bc.LastHash)
	if block.Height > lastBlock.Height {
		bc.DB.SetData([]byte(lastHashKey), block.Hash)
		bc.LastHash = block.Hash
		fmt.Printf("🔗 Đã đồng bộ Block #%d từ Peer\n", block.Height)
	}
}

// GetBlock: Tìm Block theo Hash
func (bc *Blockchain) GetBlock(hash []byte) (*Block, error) {
	data, err := bc.DB.GetData(hash)
	if err != nil {
		return nil, err
	}
	return DeserializeBlock(data), nil
}

// GetBestHeight: Lấy chiều cao cao nhất hiện tại (Phục vụ P2P handshake)
func (bc *Blockchain) GetBestHeight() int {
	lastBlock, err := bc.GetBlock(bc.LastHash)
	if err != nil {
		return 0
	}
	return lastBlock.Height
}

// GetBlockHashes: Lấy tất cả Hash trong chuỗi (Để gửi cho thằng khác khi nó xin)
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var hashes [][]byte
	currentHash := bc.LastHash

	for {
		block, err := bc.GetBlock(currentHash)
		if err != nil {
			break
		}

		hashes = append(hashes, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break // Đã đến Genesis Block
		}
		currentHash = block.PrevBlockHash
	}

	return hashes
}

func (bc *Blockchain) GetAllBlock() []*Block {
	var blocks []*Block
	currentHash := bc.LastHash

	for {
		block, err := bc.GetBlock(currentHash)
		if err != nil {
			break
		}
		blocks = append(blocks, block)

		if len(block.PrevBlockHash) == 0 {
			break
		}
		currentHash = block.PrevBlockHash
	}

	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	return blocks
}

func (bc *Blockchain) GetBlockAffter(startHash []byte) []*Block {
	var blocks []*Block
	currentHash := bc.LastHash

	for {
		if string(currentHash) == string(startHash) {
			break
		}
		block, err := bc.GetBlock(currentHash)
		if err != nil {
			break
		}
		blocks = append(blocks, block)

		if len(block.PrevBlockHash) == 0 {
			break
		}
		currentHash = block.PrevBlockHash
	}

	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	return blocks
}
