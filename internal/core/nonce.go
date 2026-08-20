package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type ChallengeInfo struct {
	Sender    string
	ExpiresAt time.Time
}

type NonceManager struct {
	mu         sync.Mutex
	challenges map[string]ChallengeInfo
}

func NewNonceManager() *NonceManager {
	return &NonceManager{
		challenges: make(map[string]ChallengeInfo),
	}
}

var NM *NonceManager = NewNonceManager()

// Tao challenge ngau nhien co han su dung 2 phut cho sender
func (nm *NonceManager) GenerateChallenge(sender string) (string, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(bytes)

	nm.challenges[nonce] = ChallengeInfo{
		Sender:    sender,
		ExpiresAt: time.Now().Add(2 * time.Minute),
	}

	return nonce, nil
}

// Kiem tra, xac thuc va xoa ngay lap tuc de chong replay attack
func (nm *NonceManager) VerifyAndConsume(sender string, nonce string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	info, exists := nm.challenges[nonce]
	if !exists {
		return fmt.Errorf("The nonce has been used or has expired.")
	}

	delete(nm.challenges, nonce)

	if info.Sender != sender {
		return fmt.Errorf("Nonce does not match the sender.")
	}

	if time.Now().After(info.ExpiresAt) {
		return fmt.Errorf("Nonce has expired.")
	}

	return nil
}
