package examples

import (
	"fmt"
	"khoai-chain/pkg/contract"
)

type UsageExamples struct {
	contract.BaseContract
}

func NewUsageExamples() *UsageExamples {
	app := &UsageExamples{}
	app.SetName([]byte("examplesgolang"))
	return app
}

type StructTest struct {
	A []byte
	B []byte
	C []byte
	D [][]byte
}

func NewTest(a, b, c, d []byte) StructTest {
	return StructTest{
		a, b, c, [][]byte{b, c, d},
	}
}

func (ue *UsageExamples) TestAdd(args [][]byte) ([]byte, error) {
	err := ue.RequireCaller("Alice")
	if err != nil {
		return nil, err
	}
	t := NewTest(args[0], args[1], args[2], args[3])
	// them logic
	return nil, ue.Save(args[0], t)
}

func (ue *UsageExamples) TestGet(args [][]byte) ([]byte, error) {
	err := ue.RequireCaller("Alice")
	if err != nil {
		return nil, err
	}

	var t StructTest

	// 1. Lấy dữ liệu từ DB
	err = ue.Get(args[0], &t)

	if err != nil {
		fmt.Printf("❌ [TestGet] Lỗi: %v\n", err)
		return nil, err
	}

	// 2. [KIỂM TRA] In ra màn hình để xem dữ liệu có chuẩn không
	fmt.Println("\n--- 🔍 KẾT QUẢ KIỂM TRA (GET) ---")
	fmt.Printf("🔑 Key: %s\n", args[0])
	fmt.Printf("📄 Field A: %s\n", string(t.A)) // Ép kiểu byte -> string để dễ đọc
	fmt.Printf("📄 Field B: %s\n", string(t.B))
	fmt.Printf("📄 Field C: %s\n", string(t.C))

	fmt.Println("📦 Field D (Mảng lồng):")
	for i, item := range t.D {
		fmt.Printf("   - Item %d: %s\n", i, string(item))
	}
	fmt.Println("---------------------------------\n")

	return t.A, nil
}
