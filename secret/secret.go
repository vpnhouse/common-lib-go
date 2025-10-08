package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

type Sealer struct {
	block cipher.Block
}

func New(secret string) *Sealer {
	hash := sha256.New()
	hash.Write([]byte(secret))
	block, _ := aes.NewCipher(hash.Sum(nil))
	return &Sealer{block: block}
}

func (s *Sealer) SealString(plainText string) string {
	return s.Seal([]byte(plainText))
}

func (s *Sealer) Seal(plainText []byte) string {
	aesgcm, _ := cipher.NewGCM(s.block)
	nonce := make([]byte, aesgcm.NonceSize())
	_, _ = rand.Read(nonce)
	ciphertext := aesgcm.Seal(nonce, nonce, plainText, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext)
}

func (s *Sealer) UnsealToString(cipherText string) (string, error) {
	v, err := s.Unseal(cipherText)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func (s *Sealer) Unseal(cipherText string) ([]byte, error) {
	cipherTextBytes, err := base64.RawURLEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}
	aesgcm, _ := cipher.NewGCM(s.block)
	var plaintext []byte
	plainText, err := aesgcm.Open(
		plaintext,
		cipherTextBytes[:aesgcm.NonceSize()],
		cipherTextBytes[aesgcm.NonceSize():],
		nil,
	)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

func SealString(secret, plainText string) string {
	return New(secret).SealString(plainText)
}

func UnsealToString(secret, cipherText string) (string, error) {
	return New(secret).UnsealToString(cipherText)
}
