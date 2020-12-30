package text

// Package text contains method to encrypt/decrypt text messages.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// ErrEmpty is an error, when encrypted/decrypted text is empty.
var ErrEmpty = errors.New("empty text")

// Encrypt encrypts text using AES cipher by a key.
func Encrypt(plainText []byte, key []byte) ([]byte, error) {
	if len(plainText) == 0 {
		return nil, ErrEmpty
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new encrypt cipher: %w", err)
	}
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("iv random generation: %w", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)
	return cipherText, nil
}

// Decrypt returns decrypted value from text by a key.
func Decrypt(cipherText []byte, key []byte) ([]byte, error) {
	if len(cipherText) == 0 {
		return nil, ErrEmpty
	}
	if len(cipherText) < aes.BlockSize {
		return nil, errors.New("invalid decryption cipher block length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new decrypt cipher: %w", err)
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)
	return cipherText, nil
}
