package core

import (
	"encoding/hex"
	"fmt"
	"khoai-chain/internal/database"
	"os"
	"time"
)

const (
	lastHashKey = "lh" // Special key to store the hash of the latest block
)

type Blockchain struct {
	LastHash string            // Hash of the latest block (the tip)
	DB       *database.Storage // Connection to the Database
}

// InitBlockchain: Initializes the blockchain.
// If DB is empty -> Create Genesis.
// If DB exists -> Load LastHash.
func InitBlockchain(db *database.Storage) *Blockchain {
	var lastHash string

	// Check if the Blockchain already exists in the DB
	if !db.HasKey([]byte(lastHashKey)) {
		fmt.Println("No existing blockchain found, creating Genesis Block...")

		// 1. Create Genesis Block
		genesis := NewGenesisBlock()

		// 2. Save Block to DB (Key=Hash, Value=Serialized)
		err := db.SetData([]byte(genesis.Hash), genesis.Serialize())
		if err != nil {
			fmt.Println("Error saving Genesis block:", err)
			os.Exit(1)
		}

		// 3. Save the LastHash pointer
		err = db.SetData([]byte(lastHashKey), []byte(genesis.Hash))
		if err != nil {
			fmt.Println("Error saving LastHash:", err)
			os.Exit(1)
		}

		lastHash = genesis.Hash
	} else {
		// If it exists, get the last hash
		var err error
		data, err := db.GetData([]byte(lastHashKey))
		if err != nil {
			fmt.Println("Error reading LastHash:", err)
			os.Exit(1)
		}
		lastHash = hex.EncodeToString(data)
		fmt.Printf("Blockchain loaded. LastHash: %x\n", lastHash)
	}

	return &Blockchain{LastHash: lastHash, DB: db}
}

// MineBlock: Packages transactions into a new block and adds it to the chain
func (bc *Blockchain) MineBlock(txs []*Transaction) *Block {
	// 1. Get the last block to know its Hash and Height
	lastBlock, err := bc.GetBlock(bc.LastHash)
	if err != nil {
		fmt.Println("Could not find the last block:", err)
		return nil
	}

	// 2. Create a new Block (Height incremented by 1)
	newBlock := NewBlock(txs, bc.LastHash, time.Now().Unix(), lastBlock.Height+1)

	// 3. Check if block already exists in DB to prevent duplicate mining
	if bc.DB.HasKey([]byte(newBlock.Hash)) {
		fmt.Println("Block already exists, skipping duplicate mine.")
		return nil
	}

	// 4. Save the new Block to DB
	err = bc.DB.SetData([]byte(newBlock.Hash), newBlock.Serialize())
	if err != nil {
		fmt.Println("Error saving new block:", err)
		return nil
	}

	// 5. Update LastHash
	err = bc.DB.SetData([]byte(lastHashKey), []byte(newBlock.Hash))
	if err != nil {
		fmt.Println("Error updating LastHash:", err)
		return nil
	}

	// 6. Update the struct in memory
	bc.LastHash = newBlock.Hash

	fmt.Printf("Mined Block #%d [%x]\n", newBlock.Height, newBlock.Hash)
	return newBlock
}

// AddBlock: This function is for P2P Sync (when receiving a Block from another peer)
func (bc *Blockchain) AddBlock(block *Block) {
	// 1. Check if block already exists to avoid duplication
	if bc.DB.HasKey([]byte(block.Hash)) {
		return
	}

	// 2. Save the block to database first
	err := bc.DB.SetData([]byte(block.Hash), block.Serialize())
	if err != nil {
		fmt.Println("Error saving block from peer:", err)
		return
	}

	// 3. Check if this block strictly extends our chain with a higher height
	lastBlock, err := bc.GetBlock(bc.LastHash)
	if err == nil && block.Height > lastBlock.Height {
		err = bc.DB.SetData([]byte(lastHashKey), []byte(block.Hash))
		if err == nil {
			bc.LastHash = block.Hash
			fmt.Printf("Synced Block #%d from Peer\n", block.Height)
		}
	}
}

// GetBlock: Find a Block by its Hash
func (bc *Blockchain) GetBlock(hash string) (*Block, error) {
	data, err := bc.DB.GetData([]byte(hash))
	if err != nil {
		return nil, err
	}
	return DeserializeBlock(data), nil
}

// GetBestHeight: Get the current highest height (for P2P handshake)
func (bc *Blockchain) GetBestHeight() int {
	lastBlock, err := bc.GetBlock(bc.LastHash)
	if err != nil {
		return 0
	}
	return lastBlock.Height
}

// GetBlockHashes: Get all Hashes in the chain (to send to another peer on request)
func (bc *Blockchain) GetBlockHashes() []string {
	var hashes []string
	currentHash := bc.LastHash

	for {
		block, err := bc.GetBlock(currentHash)
		if err != nil {
			break
		}

		hashes = append(hashes, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break // Reached Genesis Block
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

func (bc *Blockchain) GetBlockAfter(startHash string) []*Block {
	var blocks []*Block
	currentHash := bc.LastHash

	for {
		if currentHash == startHash {
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
