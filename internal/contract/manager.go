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
	currentSender string
	permission    string

	OnBlockMined func(block *core.Block)
	stagingState map[string][]byte
}

func NewManager(chain *core.Blockchain) *ContractManager {
	return &ContractManager{
		router:  &Router{},
		Chain:   chain,
		Mempool: core.NewMempool(),
	}
}

func (cm *ContractManager) PutState(key string, value []byte) {
	cm.stagingState[key] = value
}

func (cm *ContractManager) GetState(key string) ([]byte, error) {
	if val, ok := cm.stagingState[key]; ok {
		return val, nil
	}
	return cm.Chain.DB.GetData([]byte(key))
}

func (cm *ContractManager) GetSender() string {
	return cm.currentSender
}

func (cm *ContractManager) Commit() error {
	for k, v := range cm.stagingState {
		var err = cm.Chain.DB.SetData([]byte(k), v)
		return err
	}

	return nil
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

func (cm *ContractManager) ExecuteOnly(sender, contractName, method string, args []string) ([]byte, error, error) {
	cm.stagingState = make(map[string][]byte)

	cm.currentSender = sender
	cm.permission = core.PublicKeyPeers[string(sender)]

	if cm.permission == "" {
		return nil, nil, fmt.Errorf("permission does not exist")
	}
	app, exists := sdk.GlobalRegistry[string(contractName)]
	if !exists {
		return nil, nil, fmt.Errorf("contract '%s' is not installed", contractName)
	}
	app.SetContext(cm)

	var result, errContract, err = cm.router.CallMethod(app, sender, method, args)
	if err != nil {
		cm.stagingState = nil
		return nil, nil, err
	}

	return result, errContract, err
}

// Execute: Run logic -> Create Tx -> Mine Block
func (cm *ContractManager) Execute(sender, contractName, method string, args []string) ([]byte, *core.Transaction, error, error) {
	result, errContract, err := cm.ExecuteOnly(sender, contractName, method, args)
	if err != nil {
		return nil, nil, errContract, err
	}

	tx := core.NewTransaction(sender, contractName, method, args, time.Now().UnixNano())
	_, err = cm.AddAndCheckMine(tx)
	if err != nil {
		return nil, nil, errContract, err
	}

	return result, tx, errContract, nil
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
