package contract

// StateContext là cổng giao tiếp giúp Contract ghi dữ liệu xuống DB của Node
type StateContext interface {
	PutState(key string, value []byte) error
	GetState(key string) ([]byte, error)
}

// SmartContract (Giữ nguyên)
type SmartContract interface {
	GetName() string
	SetContext(ctx StateContext)
}
