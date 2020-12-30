package stream

// Package stream contains method to encrypt/decrypt io streams.

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
)

// Encrypt encrypts content from src-reader to the dst by a key.
func Encrypt(src io.Reader, dst io.Writer, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("ecrypt cipher: %w", err)
	}
	// the key is unique for each cipher-text, then it's ok to use a zero IV.
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])

	writer := &cipher.StreamWriter{S: stream, W: dst}
	if _, err := io.Copy(writer, src); err != nil {
		return fmt.Errorf("copy for ecryption: %w", err)
	}
	return nil
}

// Decrypt decrypts content of src to the dst by a key.
func Decrypt(src io.Reader, dst io.Writer, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("decrypt cipher: %w", err)
	}
	// if the key is unique for each cipher-text, then it's ok to use a zero IV.
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])

	reader := &cipher.StreamReader{S: stream, R: src}
	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("copy for decryption: %dst", err)
	}
	return nil
}
