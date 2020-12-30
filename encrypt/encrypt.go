package encrypt

// Package encrypt contains methods to encrypt/decrypt texts and files.

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"

	"github.com/z0rr0/send/encrypt/stream"
	"github.com/z0rr0/send/encrypt/text"
)

const (
	// saltSize is random of salt.
	saltSize = 128
	// fileNameSize is used for storage file name.
	fileNameSize = 64
	// fileCreateAttempts is a number of attempts to create new file with unique name.
	fileCreateAttempts = 10
	// pbkdf2Iter is number of pbkdf2 iterations.
	pbkdf2Iter = 65536
	// key length for AES-256.
	aesKeyLength = 32
	// hashLength is length of file hash.
	hashLength = 32
)

// ErrSecret is an error when the secret hash is incorrect.
var ErrSecret = errors.New("failed secret")

// Msg is struct with base parameter/results of encryption/decryption.
type Msg struct {
	Salt  string
	Value string
	Hash  string
	s     []byte
	v     []byte
	h     []byte
}

func (m *Msg) encode(withValue bool) {
	m.Salt = hex.EncodeToString(m.s)
	m.Hash = hex.EncodeToString(m.h)
	if withValue {
		m.Value = hex.EncodeToString(m.v)
	}
}

func (m *Msg) decode(withValue bool) error {
	b, err := hex.DecodeString(m.Salt)
	if err != nil {
		return fmt.Errorf("hex decode salt: %w", err)
	}
	m.s = b

	b, err = hex.DecodeString(m.Hash)
	if err != nil {
		return fmt.Errorf("hex decode hash: %w", err)
	}
	m.h = b

	if withValue {
		b, err = hex.DecodeString(m.Value)
		if err != nil {
			return fmt.Errorf("hex decode value: %w", err)
		}
		m.v = b
	}
	return nil
}

// random returns n-random bytes.
func random(n int) ([]byte, error) {
	result := make([]byte, n)
	_, err := rand.Read(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// createFile creates a new file with random name inside base path.
func createFile(base string) (*os.File, error) {
	for i := 0; i < fileCreateAttempts; i++ {
		value, err := random(fileNameSize)
		if err != nil {
			return nil, fmt.Errorf("random file name: %w", err)
		}
		fullPath := filepath.Join(base, hex.EncodeToString(value))
		f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			if !os.IsExist(err) {
				// unexpected error
				return nil, fmt.Errorf("random file creation: %w", err)
			}
			// do new attempt
		} else {
			return f, nil
		}
	}
	return nil, fmt.Errorf("can not create new file after %d attempts", fileCreateAttempts)
}

// Salt returns random bytes.
func Salt() ([]byte, error) {
	salt, err := random(saltSize)
	if err != nil {
		return nil, fmt.Errorf("read rand: %w", err)
	}
	return salt, nil
}

// Hash returns SHA3 ShakeSum256 check sum with length 32 bit.
func Hash(data []byte) []byte {
	b := make([]byte, hashLength)
	sha3.ShakeSum256(b, data)
	return b
}

// Key calculates and returns secret key and its SHA512 hash.
func Key(secret string, salt []byte) ([]byte, []byte) {
	key := pbkdf2.Key([]byte(secret), salt, pbkdf2Iter, aesKeyLength, sha3.New512)
	return key, Hash(append(key, salt...))
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
	m.encode(true)
	return m, nil
}

// DecryptText returns decrypted value from Msg.Value using the secret.
// Salt in m.Salt is expected
func DecryptText(secret string, m *Msg) (string, error) {
	err := m.decode(true)
	if err != nil {
		return "", err
	}
	key, hash := Key(secret, m.s)
	if !hmac.Equal(hash, m.h) {
		return "", ErrSecret
	}
	plainText, err := text.Decrypt(m.v, key)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

// File encrypts content from src to a new file using the secret.
// Salt and key hash are returned as Msg.Salt and Msg.Hash.
// The name if new file will be stored in m.Value.
func File(secret string, src io.Reader, base string) (*Msg, error) {
	salt, err := Salt()
	if err != nil {
		return nil, err
	}
	dst, err := createFile(base)
	if err != nil {
		return nil, fmt.Errorf("open file for ecryption: %w", err)
	}
	key, h := Key(secret, salt)
	err = stream.Encrypt(src, dst, key)
	if err != nil {
		return nil, err
	}
	m := &Msg{s: salt, h: h, Value: dst.Name()}
	m.encode(false)
	return m, dst.Close()
}

// DecryptFile writes decrypted content of file with path from Msg.Value,
// checking Msg.Hash to dst using the secret and Msg.Salt.
func DecryptFile(secret string, m *Msg, dst io.Writer) error {
	err := m.decode(false)
	if err != nil {
		return err
	}
	src, err := os.Open(m.Value)
	if err != nil {
		return fmt.Errorf("open file for decryption: %w", err)
	}
	key, hash := Key(secret, m.s)
	if !hmac.Equal(hash, m.h) {
		return ErrSecret
	}
	err = stream.Decrypt(src, dst, key)
	if err != nil {
		return err
	}
	return src.Close()
}
