package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
)

// Identity luu tru cap khoa thuoc ve mot thuc the tren mang
type Identity struct {
	PublicKey  ed25519.PublicKey  `yaml:"public_key"`
	PrivateKey ed25519.PrivateKey `yaml:"-"`
}

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
