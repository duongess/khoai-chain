package contract

import (
	"fmt"
	"khoai-chain/internal/core"
)

type ContractManager struct {
	contracts map[string]SmartContract
	router    *Router // Thêm bộ định tuyến
	Chain     *core.Blockchain
	Mempool   *core.Mempool
}

func NewManager(chain *core.Blockchain) *ContractManager {
	return &ContractManager{
		contracts: make(map[string]SmartContract),
		router:    &Router{},
		Chain:     chain,
		Mempool:   core.NewMempool(),
	}
}

func (cm *ContractManager) PutState(key []byte, value []byte) error {
	return cm.Chain.DB.SetData(key, value)
}

func (cm *ContractManager) GetState(key []byte) ([]byte, error) {
	return cm.Chain.DB.GetData(key)
}

// RegisterApp: Đăng ký App mới
func (cm *ContractManager) RegisterApp(app SmartContract) {
	name := app.GetName()
	cm.contracts[string(name)] = app
	fmt.Printf("📦 Đã load Smart Contract: %s\n", name)
}

// Execute: Chạy logic -> Tạo Tx -> Đào Block
func (cm *ContractManager) Execute(contractName []byte, method []byte, args [][]byte) ([]byte, error) {
	// 1. Tìm App
	app, exists := cm.contracts[string(contractName)]
	if !exists {
		return nil, fmt.Errorf("contract '%s' chưa được cài đặt", contractName)
	}
	app.SetContext(cm)

	// 2. CHẠY THỬ LOGIC (Simulation)
	// Để xem có lỗi gì không (ví dụ: kho hết hàng, sai tham số...)
	result, err := cm.router.CallMethod(app, method, args)
	if err != nil {
		fmt.Println("❌ Lỗi Contract (Reverted):", err)
		return nil, err // Lỗi thì trả về luôn, không lưu transaction
	}

	// 3. NẾU THÀNH CÔNG -> TẠO TRANSACTION (Đóng gói)
	// (Giả sử người gửi là AdminLocal - sau này lấy từ API Auth)
	tx := core.NewTransaction(
		[]byte("Local_Admin"),
		contractName,
		method,
		args,
	)

	// 4. ĐÀO BLOCK (Instant Mining)
	fmt.Println("⛏️  Đang đóng gói giao dịch...")
	txsToMine, ready := cm.Mempool.Add(tx)
	if ready {
		fmt.Printf("🚀 Đủ 10 giao dịch -> Kích hoạt ĐÀO BLOCK!\n")
		newBlock := cm.Chain.MineBlock(txsToMine)
		if newBlock == nil {
			return nil, fmt.Errorf("lỗi đào block")
		}
	}

	// Trả về kết quả thực thi của Smart Contract
	return result, nil
}
