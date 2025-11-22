package contract

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// BaseContract: Lớp cha (cho khách hàng kế thừa)
type BaseContract struct {
	Name []byte
	Ctx  StateContext
}

// SetName: Đặt tên cho contract
func (b *BaseContract) SetName(n []byte) {
	b.Name = n
}

func (b *BaseContract) SetContext(ctx StateContext) {
	b.Ctx = ctx
}

// GetName: Lấy tên (Impl mặc định cho Interface)
func (b *BaseContract) GetName() []byte {
	return b.Name
}

// Save: Lưu struct bất kỳ xuống DB dưới dạng TOML
func (b *BaseContract) Save(key []byte, data interface{}) error {
	if b.Ctx == nil {
		return fmt.Errorf("chưa kết nối Database (Ctx is nil)")
	}

	// 1. Chuyển Struct -> TOML Bytes
	bytesData, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("lỗi tạo TOML: %v", err)
	}

	// 2. Tạo Key định danh (Namespace): "tencontract_key"
	// VD: "vericon_kho_hang_01"
	realKey := fmt.Sprintf("%s_%s", b.Name, key)

	// 3. Lưu xuống DB
	return b.Ctx.PutState([]byte(realKey), bytesData)
}

// Get: Đọc từ DB và đổ dữ liệu vào struct (target phải là con trỏ)
func (b *BaseContract) Get(key []byte, target interface{}) error {
	if b.Ctx == nil {
		return fmt.Errorf("chưa kết nối Database")
	}

	// 1. Tạo Key định danh
	realKey := fmt.Sprintf("%s_%s", b.Name, key)

	// 2. Lấy dữ liệu thô từ DB
	bytesData, err := b.Ctx.GetState([]byte(realKey))
	if err != nil {
		return err // Không tìm thấy hoặc lỗi DB
	}

	// 3. Chuyển TOML Bytes -> Struct
	err = toml.Unmarshal(bytesData, target)
	if err != nil {
		return fmt.Errorf("dữ liệu trong DB không phải chuẩn TOML: %v", err)
	}

	return nil
}

func (b *BaseContract) RequireCaller(allowed ...string) error {
	sender := b.Ctx.GetSender()
	for _, p := range allowed {
		if p == string(sender) {
			return nil
		}
	}
	return fmt.Errorf("⛔ Truy cập bị từ chối! Sender '%s' không có quyền", sender)
}
