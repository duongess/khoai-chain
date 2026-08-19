package contract

import (
	"fmt"
	"khoai-chain/internal/core"
	sdk "khoai-chain/pkg/contract"
)

type ContractManager struct {
	contracts     map[string]sdk.SmartContract
	router        *Router // Add router
	Chain         *core.Blockchain
	Mempool       *core.Mempool
	currentSender []byte
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

// RegisterApp: Register a new App
func (cm *ContractManager) RegisterApp(app sdk.SmartContract) {
	name := app.GetName()
	sdk.GlobalRegistry[string(name)] = app
	fmt.Printf("Smart Contract loaded: %s\n", name)
}

// Execute: Run logic -> Create Tx -> Mine Block
func (cm *ContractManager) Execute(sender, contractName, method []byte, args [][]byte) ([]byte, error) {
	// 1. Find App
	cm.currentSender = sender
	app, exists := sdk.GlobalRegistry[string(contractName)]
	if !exists {
		return nil, fmt.Errorf("contract '%s' is not installed", contractName)
	}
	app.SetContext(cm)

	// 2. RUN LOGIC (Simulation)
	result, err := cm.router.CallMethod(app, sender, method, args)
	if err != nil {
		fmt.Println("Contract Error (Reverted):", err)
		return nil, err // If there's an error, return immediately, don't save the transaction
	}

	// 3. IF SUCCESSFUL -> CREATE TRANSACTION (Package)
	// (Assuming sender is AdminLocal - later get from Auth API)
	tx := core.NewTransaction(
		sender,
		contractName,
		method,
		args,
	)

	// 4. MINE BLOCK (Instant Mining)
	fmt.Println("Packaging transaction...")
	txsToMine, ready := cm.Mempool.Add(tx)
	if ready {
		fmt.Printf("10 transactions reached -> Activating BLOCK MINING!\n")
		newBlock := cm.Chain.MineBlock(txsToMine)
		if newBlock == nil {
			return nil, fmt.Errorf("error mining block")
		}
	}

	// Return the execution result of the Smart Contract
	return result, nil
}
