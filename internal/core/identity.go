package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Identity luu tru cap khoa thuoc ve mot thuc the tren mang
type Identity struct {
	PublicKey  ed25519.PublicKey  `yaml:"public_key"`
	PrivateKey ed25519.PrivateKey `yaml:"-"`
}

var PublicKey []byte
var Permission string

// GenerateIdentity tao ra mot cap khoa moi bang Ed25519
func GenerateIdentity() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

func LoadIdentity(keyDir string, permission string) error {
	pubPath := filepath.Join(keyDir, "public.pub")
	pubHex, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("don't read public key: %w", err)
	}

	pubBytes, err := hex.DecodeString(string(pubHex))
	if err != nil {
		return fmt.Errorf("error decoding public key: %w", err)
	}

	PublicKey = pubBytes
	Permission = permission

	return nil

}

// Verify dung Public Key de xac thuc chu ky cua mot thong diep
func VerifySignature(pubKey ed25519.PublicKey, message []byte, signature []byte) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return errors.New("kich thuoc public key khong hop le")
	}

	if len(signature) != ed25519.SignatureSize {
		return errors.New("kich thuoc chu ky khong hop le")
	}

	isValid := ed25519.Verify(pubKey, message, signature)
	if !isValid {
		return errors.New("chu ky khong khop voi du lieu hoac public key")
	}

	return nil
}
