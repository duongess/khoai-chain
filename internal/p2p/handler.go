package p2p

import (
	"encoding/json"
	"fmt"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core" // Import để dùng struct Block nếu cần
)

// Hàm này trả về ([]byte, error).
// []byte là dữ liệu JSON ĐÃ ĐÓNG GÓI SẴN để gửi lại cho bên kia.
func HandleMessage(payload []byte, manager *contract.ContractManager) ([]byte, error) {
	fmt.Printf("Đang xử lý dữ liệu: %s\n", string(payload))

	var msg CommandMessage
	// Nếu JSON hỏng ngay từ đầu -> Trả về lỗi dạng JSON cho Client biết
	if err := json.Unmarshal(payload, &msg); err != nil {
		// Tự đóng gói lỗi JSON tại đây
		resp := ResponseMessage{Status: "Error", Error: "Invalid JSON format"}
		return json.Marshal(resp)
	}

	switch msg.Type {

	// --- TRƯỜNG HỢP 1: CLIENT GỬI LỆNH (TRẢ VỀ ResponseMessage) ---
	case MsgExecute:
		var argsBytes [][]byte
		for _, arg := range msg.Args {
			argsBytes = append(argsBytes, []byte(arg))
		}

		// Gọi Contract
		result, err := manager.Execute(
			[]byte(msg.Sender),
			[]byte(msg.Contract),
			[]byte(msg.Function),
			argsBytes,
		)

		// Tự đóng gói ResponseMessage tại đây
		var resp ResponseMessage
		if err != nil {
			resp = ResponseMessage{Status: "Error", Error: err.Error()}
		} else {
			resp = ResponseMessage{Status: "Success", Result: string(result)}
		}

		// Trả về byte đã marshal
		return json.Marshal(resp)

	case MsgSendChain:
		var resp SendBlocksRequest
		if err := json.Unmarshal(payload, &resp); err != nil {
			return nil, err
		}

		fmt.Printf("📥 Nhận %d blocks để đồng bộ...\n", len(resp.Blocks))
		for _, block := range resp.Blocks {
			manager.Chain.AddBlock(block)
		}
		return nil, nil

	case MsgGetChain:
		var req GetBlocksRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, err
		}

		var commonHash []byte
		// Tìm điểm giao nhau
		for _, hash := range req.Hashes {
			if manager.Chain.DB.HasKey(hash) {
				commonHash = hash
				break
			}
		}

		var blocksToSend []*core.Block
		if commonHash == nil {
			fmt.Println("⚠️ Không tìm thấy điểm giao nhau (Full Sync)")
			blocksToSend = manager.Chain.GetAllBlock()
		} else {
			fmt.Printf("📍 Giao nhau tại: %x\n", commonHash)
			blocksToSend = manager.Chain.GetBlockAffter(commonHash)
		}

		resp := SendBlocksRequest{
			Type:   MsgSendChain,
			Blocks: blocksToSend,
		}
		return json.Marshal(resp)
	}

	errResp := ResponseMessage{Status: "Error", Error: "Unknown command"}
	return json.Marshal(errResp)
}
