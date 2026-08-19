package contract

var GlobalRegistry = make(map[string]SmartContract)

// StateContext
type StateContext interface {
	PutState(key []byte, value []byte) error
	GetState(key []byte) ([]byte, error)
	GetSender() []byte
}

// SmartContract
type SmartContract interface {
	GetName() []byte
	SetContext(ctx StateContext)
}
