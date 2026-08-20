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
	serverAddress := "0.0.0.0:8080"

	// Nhap truc tiep Hex string cua Private Key va Public Key de test
	// Vi du: tao cap khoa Ed25519 va dien vao day
	privKeyHex := "105daa375f69aca1a4750da1cda05cfd996218333968a4a827360a5cb6dae6ed9737530538514654c8e3f650b0c02c9535e61d33958e25b65e3a5b5a0506c010"
	pubKeyHex := "9737530538514654c8e3f650b0c02c9535e61d33958e25b65e3a5b5a0506c010"

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
		Args:     []string{"1", "1"},
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
