package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// 1. Kết nối đến Server (Đảm bảo Server đang chạy port 9000)
	serverAddress := "localhost:9001"
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("❌ Không kết nối được server:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Đã kết nối %s. Hãy nhập JSON lệnh và nhấn Enter.\n", serverAddress)
	fmt.Println("Ví dụ: {\"type\":\"EXECUTE\", \"contract\":\"examplesgolang\", \"function\":\"TestAdd\", \"args\":[\"a\",\"b\"]}")
	fmt.Println("---------------------------------------------------------------")

	// 2. Vòng lặp: Đọc bàn phím -> Gửi -> Nhận phản hồi
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Client > ")

		// Đọc dòng bạn nhập
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		// Gửi qua TCP (thêm \n để server biết hết câu)
		fmt.Fprintf(conn, input+"\n")

		// Đọc phản hồi từ Server
		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Mất kết nối server.")
			return
		}

		fmt.Print("Server < " + response)
	}
}
