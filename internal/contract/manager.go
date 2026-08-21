package contract

import (
	"fmt"
	"khoai-chain/internal/core"
	sdk "khoai-chain/pkg/contract"
	"time"
)

type ContractManager struct {
	contracts     map[string]sdk.SmartContract
	router        *Router // Add router
	Chain         *core.Blockchain
	Mempool       *core.Mempool
	currentSender []byte
	permission    string

	OnBlockMined func(block *core.Block)
}

func NewManager(chain *core.Blockchain) *ContractManager {
	return &ContractManager{
		router:  &Router{},
		Chain:   chain,
		Mempool: core.NewMempool(),
	}
}

func (cm *ContractManager) PutState(key []byte, value []byte) error {
	return cm.Chain.DB.SetData(key, value)
}

func (cm *ContractManager) GetState(key []byte) ([]byte, error) {
	return cm.Chain.DB.GetData(key)
}

func (cm *ContractManager) GetSender() []byte {
	return cm.currentSender
}

func (cm *ContractManager) GetPermission() string {
	return cm.permission
}

// RegisterApp: Register a new App
func (cm *ContractManager) RegisterApp(app sdk.SmartContract) {
	name := app.GetName()
	sdk.GlobalRegistry[string(name)] = app
	fmt.Printf("Smart Contract loaded: %s\n", name)
}

func (cm *ContractManager) ExecuteOnly(sender, contractName, method []byte, args [][]byte) ([]byte, error) {
	cm.currentSender = sender
	cm.permission = core.PublicKeyPeers[string(sender)]

	if cm.permission == "" {
		return nil, fmt.Errorf("permission does not exist")
	}
	app, exists := sdk.GlobalRegistry[string(contractName)]
	if !exists {
		return nil, fmt.Errorf("contract '%s' is not installed", contractName)
	}
	app.SetContext(cm)

	// Chỉ chạy logic contract để thay đổi State DB, không tạo tx, không mine block
	return cm.router.CallMethod(app, sender, method, args)
}

// Execute: Run logic -> Create Tx -> Mine Block
func (cm *ContractManager) Execute(sender, contractName, method []byte, args [][]byte) ([]byte, error) {
	result, err := cm.ExecuteOnly(sender, contractName, method, args)
	if err != nil {
		return nil, err
	}

	tx := core.NewTransaction(sender, contractName, method, args, time.Now().UnixNano())
	_, err = cm.AddAndCheckMine(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cm *ContractManager) AddAndCheckMine(tx *core.Transaction) (*core.Block, error) {
	txsToMine, ready := cm.Mempool.Add(tx)
	if ready {
		fmt.Printf("10 transactions reached -> Activating BLOCK MINING!\n")
		newBlock := cm.Chain.MineBlock(txsToMine)
		if newBlock == nil {
			return nil, fmt.Errorf("error mining block")
		}

		if cm.OnBlockMined != nil {
			cm.OnBlockMined(newBlock)
		}
		return newBlock, nil
	}
	return nil, nil
}
