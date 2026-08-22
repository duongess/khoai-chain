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
		lastHash, _ := hex.DecodeString(string(data))
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

// AddBlock is for P2P Sync when receiving a single Block from another peer.
// It only accepts blocks that directly extend the current tip — this
// prevents a block from a different chain/genesis lineage from silently
// being adopted and fragmenting the DB into disconnected branches.
func (bc *Blockchain) AddBlock(block *Block) error {
	if bc.DB.HasKey([]byte(block.Hash)) {
		return nil // already have it
	}

	if block.PrevBlockHash != bc.LastHash {
		return fmt.Errorf("rejected block %s: does not extend tip %s (prev=%s)",
			block.Hash, bc.LastHash, block.PrevBlockHash)
	}

	if err := bc.DB.SetData([]byte(block.Hash), block.Serialize()); err != nil {
		return fmt.Errorf("error saving block from peer: %w", err)
	}

	if err := bc.DB.SetData([]byte(lastHashKey), []byte(block.Hash)); err != nil {
		return fmt.Errorf("error updating LastHash: %w", err)
	}

	bc.LastHash = block.Hash
	fmt.Printf("Synced Block #%d from Peer\n", block.Height)
	return nil
}

// AddBlocks atomically validates and adopts a batch of blocks received
// during sync. It requires the batch to form a contiguous chain, connect
// to a block already present locally (a genuine common ancestor — not
// just "any hash we happen to recognize"), and share the same genesis.
// Nothing is written until the whole batch passes validation.
func (bc *Blockchain) AddBlocks(blocks []*Block) error {
	if len(blocks) == 0 {
		return nil
	}

	localGenesis, err := bc.GenesisHash()
	if err != nil {
		return fmt.Errorf("could not determine local genesis: %w", err)
	}

	// Internal continuity of the batch itself.
	for i := 1; i < len(blocks); i++ {
		if blocks[i].PrevBlockHash != blocks[i-1].Hash {
			return fmt.Errorf("broken chain link in sync batch at index %d", i)
		}
	}

	// The batch must attach to a block we actually have — i.e. a real
	// common ancestor — not be a disjoint branch we're about to graft on.
	first := blocks[0]
	if !bc.DB.HasKey([]byte(first.PrevBlockHash)) && first.PrevBlockHash != "" {
		return fmt.Errorf("batch does not connect to local chain: unknown ancestor %s", first.PrevBlockHash)
	}

	// Walk the batch back to ITS root to confirm shared genesis.
	// If the batch's PrevBlockHash chain bottoms out locally rather than
	// at "", fetch the local ancestor and continue the walk from there.
	rootHash := first.PrevBlockHash
	if rootHash != "" {
		ancestor, err := bc.GetBlock(rootHash)
		if err != nil {
			return fmt.Errorf("could not load common ancestor %s: %w", rootHash, err)
		}
		for ancestor.PrevBlockHash != "" {
			ancestor, err = bc.GetBlock(ancestor.PrevBlockHash)
			if err != nil {
				return fmt.Errorf("broken local chain while tracing genesis: %w", err)
			}
		}
		if ancestor.Hash != localGenesis {
			return fmt.Errorf("rejected sync batch: genesis mismatch (got %s, want %s)", ancestor.Hash, localGenesis)
		}
	}
	// else: batch attaches at the root itself; first.Hash IS the genesis
	// candidate, so it must equal our own genesis exactly.
	if rootHash == "" && first.Hash != localGenesis {
		return fmt.Errorf("rejected sync batch: foreign genesis %s (want %s)", first.Hash, localGenesis)
	}

	// All validated — commit.
	newTip := bc.LastHash
	newHeight := int64(-1)
	if lb, err := bc.GetBlock(bc.LastHash); err == nil {
		newHeight = int64(lb.Height)
	}

	for _, block := range blocks {
		if bc.DB.HasKey([]byte(block.Hash)) {
			continue
		}
		if err := bc.DB.SetData([]byte(block.Hash), block.Serialize()); err != nil {
			return fmt.Errorf("error saving synced block %s: %w", block.Hash, err)
		}
		if int64(block.Height) > newHeight {
			newTip = block.Hash
			newHeight = int64(block.Height)
		}
	}

	if newTip != bc.LastHash {
		if err := bc.DB.SetData([]byte(lastHashKey), []byte(newTip)); err != nil {
			return fmt.Errorf("error updating LastHash after sync: %w", err)
		}
		bc.LastHash = newTip
	}

	fmt.Printf("Synchronized %d blocks; tip now #%d [%s]\n", len(blocks), newHeight, newTip)
	return nil
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

func (bc *Blockchain) GenesisHash() (string, error) {
	hash := bc.LastHash
	for {
		block, err := bc.GetBlock(hash)
		if err != nil {
			return "", err
		}
		if block.PrevBlockHash == "" {
			return block.Hash, nil
		}
		hash = block.PrevBlockHash
	}
}
