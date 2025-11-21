package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
)

func HandleMessage(payload []byte, manager *contract.ContractManager) ([]byte, error) {
	fmt.Printf("Đang xử lý dữ liệu: %s\n", string(payload))
	var msg CommandMessage
	// Parse JSON string bình thường
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}
	if msg.Type == "EXECUTE" {
		// --- BƯỚC CHUYỂN ĐỔI (QUAN TRỌNG) ---
		var argsBytes [][]byte
		for _, arg := range msg.Args {
			argsBytes = append(argsBytes, []byte(arg))
		}

		// Gọi Contract
		result, err := manager.Execute(
			[]byte(msg.Contract),
			[]byte(msg.Function),
			argsBytes,
		)

		if err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}

	return nil, fmt.Errorf("unknow commant")
}
