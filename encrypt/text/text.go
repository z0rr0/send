package text

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// EmptyError is an error, when encrypted/decrypted text is empty.
var EmptyError = errors.New("empty text")

// Encrypt encrypts text using AES cipher by a key.
func Encrypt(value string, key []byte) (string, error) {
	if value == "" {
		return "", EmptyError
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new encrypt cipher: %w", err)
	}
	plainText := []byte(value)
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("iv random generation: %w", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)
	return hex.EncodeToString(cipherText), nil
}

// Decrypt returns decrypted value from text by a key.
func Decrypt(value string, key []byte) (string, error) {
	if value == "" {
		return "", EmptyError
	}
	cipherText, err := hex.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decrypt hex decode: %w", err)
	}
	if len(cipherText) < aes.BlockSize {
		return "", errors.New("invalid decryption cipher block length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new decrypt cipher: %w", err)
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)
	return string(cipherText), nil
}
