package contract

// StateContext là cổng giao tiếp giúp Contract ghi dữ liệu xuống DB của Node
type StateContext interface {
	PutState(key []byte, value []byte) error
	GetState(key []byte) ([]byte, error)
}

// SmartContract (Giữ nguyên)
type SmartContract interface {
	GetName() []byte
	SetContext(ctx StateContext)
}
