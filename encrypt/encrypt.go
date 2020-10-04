package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

const (
	// saltSize is random salt, also used for storage file name
	saltSize = 128
	// pbkdf2Iter is number of pbkdf2 iterations
	pbkdf2Iter = 32768
	// key length for AES-256
	aesKeyLength = 32
	// hashLength is length of file hash.
	hashLength = 32
)

func Salt() ([]byte, error) {
	salt := make([]byte, saltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("read rand: %w", err)
	}
	return salt, nil
}

// Key calculates and returns secret key and its SHA512 hash.
func Key(secret string, salt []byte) ([]byte, []byte) {
	key := pbkdf2.Key([]byte(secret), salt, pbkdf2Iter, aesKeyLength, sha3.New512)
	b := make([]byte, hashLength)
	sha3.ShakeSum256(b, append(key, salt...))
	return key, b
}

// Text encrypts text using AES cipher by a key.
func Text(text string, key []byte) (string, error) {
	if text == "" {
		return "", errors.New("encrypt empty text")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new encrypt cipher: %w", err)
	}
	plainText := []byte(text)
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("iv random generation: %w", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)
	return hex.EncodeToString(cipherText), nil
}

// DecryptText returns decrypted value from text by a key.
func DecryptText(text string, key []byte) (string, error) {
	if text == "" {
		return "", errors.New("decrypt empty text")
	}
	cipherText, err := hex.DecodeString(text)
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

// File encrypts content from inFile to new file with name fileName by a key.
func File(inFile io.Reader, fileName string, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("new file ecrypt cipher: %w", err)
	}
	// the key is unique for each cipher-text, then it's ok to use a zero IV.
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])
	outFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open file for ecryption: %w", err)
	}
	writer := &cipher.StreamWriter{S: stream, W: outFile}
	// copy the input file to the output file, encrypting as we go.
	if _, err := io.Copy(writer, inFile); err != nil {
		return fmt.Errorf("copy for ecryption: %w", err)
	}
	return outFile.Close()
}

// DecryptFile writes decrypted content of file fileName to w by a key.
func DecryptFile(w io.Writer, fileName string, key []byte) error {
	inFile, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("open file for decryption: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// if the key is unique for each cipher-text, then it's ok to use a zero IV.
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])

	reader := &cipher.StreamReader{S: stream, R: inFile}
	// copy the input file to the output file, decrypting as we go.
	if _, err := io.Copy(w, reader); err != nil {
		return fmt.Errorf("copy for decryption: %w", err)
	}
	return inFile.Close()
}
