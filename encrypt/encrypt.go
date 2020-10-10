package encrypt

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"

	"github.com/z0rr0/send/encrypt/stream"
	"github.com/z0rr0/send/encrypt/text"
)

const (
	// saltSize is random s, also used for storage file name
	saltSize = 128
	// pbkdf2Iter is number of pbkdf2 iterations
	pbkdf2Iter = 65536
	// key length for AES-256
	aesKeyLength = 32
	// hashLength is length of file hash.
	hashLength = 32
)

// Msg is struct with base parameter/results of encryption/decryption.
type Msg struct {
	Salt  string
	Value string
	Hash  string
	s     []byte
	v     []byte
	h     []byte
}

func (m *Msg) encode() {
	m.Salt = hex.EncodeToString(m.s)
	m.Hash = hex.EncodeToString(m.h)
	m.Value = hex.EncodeToString(m.v)
}

func (m *Msg) decode() error {
	b, err := hex.DecodeString(m.Salt)
	if err != nil {
		return fmt.Errorf("hex decode s: %w", err)
	}
	m.s = b

	b, err = hex.DecodeString(m.Hash)
	if err != nil {
		return fmt.Errorf("hex decode hash: %w", err)
	}
	m.h = b

	b, err = hex.DecodeString(m.Value)
	if err != nil {
		return fmt.Errorf("hex decode value: %w", err)
	}
	m.v = b
	return nil
}

// Salt returns random s.
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

// Text encrypts plaintText using the secret.
// Cipher message will be returned as Msg.Value.
func Text(secret, plainText string) (*Msg, error) {
	salt, err := Salt()
	if err != nil {
		return nil, err
	}
	key, h := Key(secret, salt)
	cipherText, err := text.Encrypt([]byte(plainText), key)
	if err != nil {
		return nil, err
	}
	m := &Msg{v: cipherText, s: salt, h: h}
	m.encode()
	return m, nil
}

// DecryptText returns decrypted value from m.Value using the secret.
func DecryptText(secret string, m *Msg) (string, error) {
	err := m.decode()
	if err != nil {
		return "", err
	}
	key, hash := Key(secret, m.s)
	if !hmac.Equal(hash, m.h) {
		return "", errors.New("failed secret")
	}
	plainText, err := text.Decrypt(m.v, key)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

// File encrypts content from src to new file with name fileName by a key.
func File(secret string, src io.Reader, fileName string) (*Msg, error) {
	salt, err := Salt()
	if err != nil {
		return nil, err
	}
	dst, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("open file for ecryption: %w", err)
	}
	key, h := Key(secret, salt)
	err = stream.Encrypt(src, dst, key)
	if err != nil {
		return nil, err
	}
	m := &Msg{s: salt, h: h}
	m.encode()
	return m, dst.Close()
}

// DecryptFile writes decrypted content of file fileName to dst by a key.
func DecryptFile(secret string, m *Msg, dst io.Writer, fileName string) error {
	err := m.decode()
	if err != nil {
		return err
	}
	src, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("open file for decryption: %w", err)
	}
	key, _ := Key(secret, m.s)
	err = stream.Decrypt(src, dst, key)
	if err != nil {
		return err
	}
	return src.Close()
}
