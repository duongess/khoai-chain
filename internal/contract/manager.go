package contract

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"khoai-chain/internal/core"
	sdk "khoai-chain/pkg/contract"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ContractMetadata struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Inputs      []FunctionParameter `json:"inputs"`
}
type FunctionParameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ContractManager struct {
	contracts     map[string]sdk.SmartContract
	router        *Router // Add router
	Chain         *core.Blockchain
	Mempool       *core.Mempool
	currentSender string
	permission    string

	OnBlockMined func(block *core.Block)
	StagingState map[string][]byte
}

func NewManager(chain *core.Blockchain) *ContractManager {
	return &ContractManager{
		router:  &Router{},
		Chain:   chain,
		Mempool: core.NewMempool(),
	}
}

func (cm *ContractManager) PutState(key string, value []byte) {
	if cm.StagingState == nil {
		cm.StagingState = make(map[string][]byte)
	}
	cm.StagingState[key] = value
}

func (cm *ContractManager) GetState(key string) ([]byte, error) {
	if val, ok := cm.StagingState[key]; ok {
		return val, nil
	}
	return cm.Chain.DB.GetData([]byte(key))
}

func (cm *ContractManager) GetSender() string {
	return cm.currentSender
}

func (cm *ContractManager) Commit() error {
	for k, v := range cm.StagingState {
		if err := cm.Chain.DB.SetData([]byte(k), v); err != nil {
			return err
		}
	}
	cm.StagingState = make(map[string][]byte)
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

func (cm *ContractManager) ExecuteOnly(payload core.TxPayload) (any, error, error) {
	cm.currentSender = payload.Sender
	cm.permission = core.PublicKeyPeers[payload.Sender]

	if cm.permission == "" {
		return nil, nil, fmt.Errorf("permission does not exist")
	}
	app, exists := sdk.GlobalRegistry[payload.Contract]
	if !exists {
		return nil, nil, fmt.Errorf("contract '%s' is not installed", payload.Contract)
	}
	app.SetContext(cm)

	var result, errContract, err = cm.router.CallMethod(app, payload.Sender, payload.Function, payload.Args)
	if err != nil {
		cm.StagingState = nil
		return nil, nil, err
	}

	return result, errContract, err
}

// Execute: Run logic -> Create Tx -> Mine Block
func (cm *ContractManager) Execute(payload core.TxPayload) (any, *core.Transaction, *core.Block, error, error) {
	result, errContract, err := cm.ExecuteOnly(payload)
	if err != nil {
		return nil, nil, nil, errContract, err
	}

	tx := core.NewTransaction(payload, time.Now().UnixNano())
	minedBlock, err := cm.AddAndCheckMine(tx)
	if err != nil {
		return nil, nil, nil, errContract, err
	}

	return result, tx, minedBlock, errContract, nil
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

func getChaincodeFiles() ([]string, error) {
	chaincodesBuildDir := filepath.Join("chaincodes")
	var validFiles []string

	entries, err := os.ReadDir(chaincodesBuildDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".go") {
			fullPath := filepath.Join(chaincodesBuildDir, entry.Name())
			validFiles = append(validFiles, fullPath)
		}
	}

	return validFiles, nil
}

func (cm *ContractManager) GetMetadataList() (any, error) {
	chaincodes, err := getChaincodeFiles()
	if err == nil {
		return nil, err
	}
	metadata := make(map[string]interface{})

	for _, srcPath := range chaincodes {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, srcPath, nil, parser.AllErrors)
		var functions []ContractMetadata

		// 1. Quet AST de tim ten Struct public dau tien lam ten Contract
		contractName := ""
		if err == nil {
			for _, decl := range node.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if ok && typeSpec.Name.IsExported() {
							if _, isStruct := typeSpec.Type.(*ast.StructType); isStruct {
								if contractName == "" {
									contractName = typeSpec.Name.Name
								}
							}
						}
					}
				}
			}
		}

		// Fallback ve ten file neu khong tim thay struct nao
		if contractName == "" {
			contractName = strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		}

		if err == nil {
			for _, decl := range node.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					continue
				}

				fnName := fn.Name.Name
				if fnName[0] < 'A' || fnName[0] > 'Z' || fnName == "Init" {
					continue
				}

				var inputs []FunctionParameter
				if fn.Type != nil && fn.Type.Params != nil {
					for _, field := range fn.Type.Params.List {
						typeName := exprToString(field.Type)

						if len(field.Names) > 0 {
							for _, name := range field.Names {
								inputs = append(inputs, FunctionParameter{
									Name: name.Name,
									Type: typeName,
								})
							}
						} else {
							inputs = append(inputs, FunctionParameter{
								Name: "arg",
								Type: typeName,
							})
						}
					}
				}

				functions = append(functions, ContractMetadata{
					Name:        fnName,
					Description: fmt.Sprintf("Auto-extracted function %s", fnName),
					Inputs:      inputs,
				})
			}
		}

		metadata[contractName] = map[string]interface{}{
			"name":      contractName,
			"functions": functions,
		}
	}

	return json.MarshalIndent(metadata, "", "  ")
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	default:
		return "unknown"
	}
}
