package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

type CommandMessage struct {
	Type      string   `json:"type"`
	Sender    string   `json:"sender"`
	Contract  string   `json:"contract"`
	Function  string   `json:"function"`
	Args      []string `json:"args"`
	Nonce     string   `json:"nonce"`
	Signature string   `json:"signature"`
}

type CommandNonce struct {
	Type   string `json:"type"`
	Sender string `json:"sender"`
}

type ResponseMessage struct {
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func BuildMessageBytes(msg CommandMessage) []byte {
	temp := msg
	temp.Signature = ""
	data, _ := json.Marshal(temp)
	return data
}

func main() {
	serverAddress := "0.0.0.0:8083"

	// Nhap truc tiep Hex string cua Private Key va Public Key de test
	// Vi du: tao cap khoa Ed25519 va dien vao day
	privKeyHex := "d52b751311d2554be0d799c6b6939ec1add4d72b681b0a840cfbc4f6d170cb0c1e6f445bff6d31a77af8cec8c8f05d12edc7f7c539e0129b196c21f860bf7021"
	pubKeyHex := "1e6f445bff6d31a77af8cec8c8f05d12edc7f7c539e0129b196c21f860bf7021"

	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		fmt.Println("Invalid private key hex:", err)
		return
	}
	privKey := ed25519.PrivateKey(privBytes)

	// --- BUOC 1: GOI LAY NONCE (CHALLENGE) ---
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Could not connect to server for nonce:", err)
		return
	}
	defer conn.Close()

	nonceReq := CommandNonce{
		Type:   "NONCE",
		Sender: pubKeyHex,
	}
	noncePayload, _ := json.Marshal(nonceReq)
	_, err = conn.Write(append(noncePayload, '\n'))
	if err != nil {
		fmt.Println("Could not send nonce request:", err)
		return
	}

	nonceRespStr, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Could not read nonce response:", err)
		return
	}

	var nonceResp ResponseMessage
	json.Unmarshal([]byte(nonceRespStr), &nonceResp)
	if nonceResp.Status != "Success" {
		fmt.Println("Failed to get nonce:", nonceResp.Error)
		return
	}
	receivedNonce := nonceResp.Result
	fmt.Println("Got Nonce/Token:", receivedNonce)

	// --- BUOC GIAO DICH: KY SO VA GOI CONTRACT ---
	cmd := CommandMessage{
		Type:     "EXECUTE",
		Sender:   pubKeyHex,
		Contract: "examplesgolang",
		Function: "CreateRawBatch",
		Args:     []string{"10", "1"},
		Nonce:    receivedNonce,
	}

	// Ky so len toan bo noi dung message (tru signature)
	msgBytes := BuildMessageBytes(cmd)
	signature := ed25519.Sign(privKey, msgBytes)
	cmd.Signature = hex.EncodeToString(signature)

	execPayload, _ := json.Marshal(cmd)
	_, err = conn.Write(append(execPayload, '\n'))
	if err != nil {
		fmt.Println("Could not send execute request:", err)
		return
	}

	execRespStr, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Could not read execute response:", err)
		return
	}

	fmt.Println("Server Response:", execRespStr)
}
