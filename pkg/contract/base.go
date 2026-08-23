package contract

import (
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

// BaseContract: Parent class (for clients to inherit)
type BaseContract struct {
	Name []byte
	Ctx  StateContext
}

func RegisterContract(c SmartContract) {
	contractName := reflect.TypeOf(c).Elem().Name()
	GlobalRegistry[contractName] = c
}

// SetName: Sets the contract name
func (b *BaseContract) SetName(n []byte) {
	b.Name = n
}

func (b *BaseContract) SetContext(ctx StateContext) {
	b.Ctx = ctx
}

// GetName: Gets the name (default implementation for the Interface)
func (b *BaseContract) GetName() []byte {
	return b.Name
}

// Save: Saves any struct to the DB as TOML
func (b *BaseContract) Save(key string, data interface{}) error {
	if b.Ctx == nil {
		return fmt.Errorf("not connected to Database (Ctx is nil)")
	}

	// 1. Convert Struct -> TOML Bytes
	bytesData, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error creating TOML: %v", err)
	}

	// 2. Create a namespace key: "contractname_key"
	// e.g., "vericon_kho_hang_01"
	realKey := fmt.Sprintf("%s_%s", b.Name, key)

	// 3. Save to DB
	b.Ctx.PutState(realKey, bytesData)
	return nil
}

// Get: Reads from DB and populates a struct (target must be a pointer)
func (b *BaseContract) Get(key string, target interface{}) error {
	if b.Ctx == nil {
		return fmt.Errorf("not connected to Database")
	}

	// 1. Create namespace key
	realKey := fmt.Sprintf("%s_%s", b.Name, key)

	// 2. Get raw data from DB
	bytesData, err := b.Ctx.GetState(realKey)
	if err != nil {
		return err // Not found or DB error
	}

	// 3. Convert TOML Bytes -> Struct
	err = toml.Unmarshal(bytesData, target)
	if err != nil {
		return fmt.Errorf("data in DB is not valid TOML: %v", err)
	}

	return nil
}

func (b *BaseContract) RequireCaller(permissions ...string) error {
	permission := b.Ctx.GetPermission()
	for _, p := range permissions {
		if p == permission {
			return nil
		}
	}
	return fmt.Errorf("Access denied! Sender '%s' does not have permission", permission)
}
