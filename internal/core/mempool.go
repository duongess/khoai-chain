package core

import (
	"fmt"
	"sync"
)

type Mempool struct {
	PendingTxs map[string][]*Transaction // Gom theo Sender
	mutex      sync.Mutex
	Threshold  int
}

func NewMempool() *Mempool {
	return &Mempool{
		PendingTxs: make(map[string][]*Transaction),
		Threshold:  10, // Gom đủ 10 mới xả
	}
}

// Add: Thêm vào kho, trả về (txs, true) nếu đủ điều kiện đào
func (mp *Mempool) Add(tx *Transaction) ([]*Transaction, bool) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	sender := string(tx.Sender)
	mp.PendingTxs[sender] = append(mp.PendingTxs[sender], tx)

	count := len(mp.PendingTxs[sender])
	fmt.Printf("📥 Mempool: [%s] đang có %d/%d giao dịch.\n", sender, count, mp.Threshold)

	if count >= mp.Threshold {
		// Cắt 10 giao dịch đầu tiên ra để đào
		txsToMine := mp.PendingTxs[sender][:mp.Threshold]
		// Giữ lại phần thừa (nếu có)
		mp.PendingTxs[sender] = mp.PendingTxs[sender][mp.Threshold:]

		return txsToMine, true
	}
	return nil, false
}
