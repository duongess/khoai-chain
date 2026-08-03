package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// 1. Connect to the Server (Ensure the Server is running on port 9000)
	serverAddress := "localhost:9000"
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("❌ Could not connect to server:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Connected to %s. Please enter a JSON command and press Enter.\n", serverAddress)
	fmt.Println("Example: {\"type\":\"EXECUTE\", \"contract\":\"examplesgolang\", \"function\":\"TestAdd\", \"args\":[\"a\",\"b\"]}")
	fmt.Println("---------------------------------------------------------------")

	// 2. Loop: Read from keyboard -> Send -> Receive response
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Client > ")

		// Read the line you entered
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		// Send via TCP (add \n so the server knows the end of the message)
		fmt.Fprintf(conn, input+"\n")

		// Read response from the Server
		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Lost connection to server.")
			return
		}

		fmt.Print("Server < " + response)
	}
}
