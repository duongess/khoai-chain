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

type IdentityMessage struct {
	Type       string
	PublicKey  []byte
	Permission string
}

var PublicKeyNode []byte
var PermissionNode string
var PublicKeyPeers = make(map[string]string)

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

func LoadIdentity(keyDir string, permissionConf string) error {
	pubPath := filepath.Join(keyDir, "public.pub")
	pubHex, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("don't read public key: %w", err)
	}

	pubBytes, err := hex.DecodeString(string(pubHex))
	if err != nil {
		return fmt.Errorf("error decoding public key: %w", err)
	}

	PublicKeyNode = pubBytes
	PermissionNode = permissionConf

	fmt.Printf("Public key: %s, have permission %s\n", hex.EncodeToString(PublicKeyNode), PermissionNode)

	return nil

}

func GetIdentityMessage() *IdentityMessage {
	return &IdentityMessage{
		Type:       "IDENTITY",
		PublicKey:  PublicKeyNode,
		Permission: PermissionNode,
	}
}

// Verify dung Public Key de xac thuc chu ky cua mot thong diep
func VerifySignature(pubKey ed25519.PublicKey, message []byte, signature []byte) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return errors.New("Invalid public key size")
	}

	if len(signature) != ed25519.SignatureSize {
		return errors.New("Invalid signature size")
	}

	isValid := ed25519.Verify(pubKey, message, signature)
	if !isValid {
		return errors.New("Signature does not match the data or public key.")
	}

	return nil
}
