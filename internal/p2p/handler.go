package p2p

import "fmt"

// HandleMessage tạm thời chưa làm gì cả, chỉ in ra thôi
func HandleMessage(payload []byte) {
	fmt.Printf("Đang xử lý dữ liệu: %s\n", string(payload))
}
