package contract

import (
	"fmt"
	"reflect"
	"strings"
)

// Router chịu trách nhiệm tìm và gọi hàm bằng Reflection
type Router struct{}

// CallMethod
// app: Đối tượng Smart Contract (VeriCon, Coin...)
// methodName: Tên hàm muốn gọi (VD: "NhapKho")
// args: Tham số đầu vào
func (r *Router) CallMethod(app interface{}, methodName []byte, args [][]byte) ([]byte, error) {
	// 1. Lấy thông tin về đối tượng App (Soi gương)
	val := reflect.ValueOf(app)

	// 2. Tìm hàm theo tên
	// Lưu ý: Hàm phải viết Hoa chữ cái đầu (Public) mới tìm thấy
	method := val.MethodByName(string(methodName))

	if !method.IsValid() {
		// Thử tìm hàm viết hoa chữ đầu (nếu user lỡ gửi chữ thường)
		method = val.MethodByName(strings.Title(string(methodName)))
		if !method.IsValid() {
			return nil, fmt.Errorf("hàm '%s' không tồn tại trong contract", methodName)
		}
	}

	// 3. Chuẩn bị tham số để gọi hàm
	// Quy ước: Hàm của User phải nhận vào ([]string)
	inputArgs := []reflect.Value{reflect.ValueOf(args)}

	// 4. GỌI HÀM (Invoke)
	results := method.Call(inputArgs)

	// 5. Xử lý kết quả trả về
	// Quy ước: Hàm của User phải trả về ([]byte, error)
	if len(results) < 2 {
		return nil, fmt.Errorf("hàm phải trả về 2 giá trị: ([]byte, error)")
	}

	// Lấy kết quả (Bytes)
	resBytes := results[0].Interface().([]byte)

	// Lấy lỗi (Error)
	errObj := results[1].Interface()
	var err error
	if errObj != nil {
		err = errObj.(error)
	}

	return resBytes, err
}
