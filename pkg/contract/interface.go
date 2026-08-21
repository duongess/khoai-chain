package contract

var GlobalRegistry = make(map[string]SmartContract)

// StateContext
type StateContext interface {
	PutState(key string, value []byte)
	GetState(key string) ([]byte, error)
	GetSender() []byte
	GetPermission() string
}

// SmartContract
type SmartContract interface {
	GetName() []byte
	SetContext(ctx StateContext)
}
