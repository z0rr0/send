package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"

	"github.com/z0rr0/send/encrypt/text"
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

// Msg is struct with base parameter/results of encryption/decryption.
type Msg struct {
	Salt     string
	Value    string
	Hash     string
	byteSalt []byte
	byteHash []byte
}

func (m *Msg) encode() {
	m.Salt = hex.EncodeToString(m.byteSalt)
	m.Hash = hex.EncodeToString(m.byteHash)
}

func (m *Msg) decode() error {
	b, err := hex.DecodeString(m.Salt)
	if err != nil {
		return fmt.Errorf("hex decode salt: %w", err)
	}
	m.byteSalt = b

	b, err = hex.DecodeString(m.Hash)
	if err != nil {
		return fmt.Errorf("hex decode hash: %w", err)
	}
	m.byteHash = b
	return nil
}

// Salt returns random salt.
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
	cipherText, err := text.Encrypt(plainText, key)
	if err != nil {
		return nil, err
	}
	m := &Msg{Value: cipherText, byteSalt: salt, byteHash: h}
	m.encode()
	return m, nil
}

// DecryptText returns decrypted value from m.Value using the secret.
func DecryptText(secret string, m *Msg) (string, error) {
	err := m.decode()
	if err != nil {
		return "", err
	}
	key, hash := Key(secret, m.byteSalt)
	if !hmac.Equal(hash, m.byteHash) {
		return "", errors.New("failed secret")
	}
	return text.Decrypt(m.Value, key)
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
