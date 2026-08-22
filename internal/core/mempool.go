package core

import (
	"fmt"
	"sync"
)

type Mempool struct {
	PendingTxs map[string][]*Transaction // Group by Sender
	mutex      sync.Mutex
	Threshold  int
}

func NewMempool() *Mempool {
	return &Mempool{
		PendingTxs: make(map[string][]*Transaction),
		Threshold:  10, // Release after collecting 10
	}
}

// Add: Adds to the pool, returns (txs, true) if ready to mine
func (mp *Mempool) Add(tx *Transaction) ([]*Transaction, bool) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	sender := tx.Payload.Sender
	mp.PendingTxs[sender] = append(mp.PendingTxs[sender], tx)

	count := len(mp.PendingTxs[sender])
	fmt.Printf("Mempool: [%s] has %d/%d transactions.\n", sender, count, mp.Threshold)

	if count >= mp.Threshold {
		// Take the first 10 transactions to mine
		txsToMine := mp.PendingTxs[sender][:mp.Threshold]
		// Keep the remainder (if any)
		mp.PendingTxs[sender] = mp.PendingTxs[sender][mp.Threshold:]

		return txsToMine, true
	}
	return nil, false
}
