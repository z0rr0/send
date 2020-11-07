package text

import (
	"bytes"
	"crypto/aes"
	"testing"
)

func buildKey(k []byte) []byte {
	const size = 32
	var n int
	key := make([]byte, size)

	if m := len(k); m < size {
		n = m
	} else {
		n = size
	}
	for i := 0; i < n; i++ {
		key[i] = k[i]
	}
	return key
}

func TestEncrypt(t *testing.T) {
	key := buildKey([]byte("abc"))
	cases := []string{
		"text",
		"other text",
		"other long long text",
	}
	for i, c := range cases {
		cb := []byte(c)
		e, err := Encrypt(cb, key)
		if err != nil {
			t.Errorf("failed ecrypt case=%d: %e", i, err)
		}
		if n := len(e); n != len(cb)+aes.BlockSize {
			t.Errorf("unexpected lenght=%d for case=%d", n, i)
		}
		// decrypt
		d, err := Decrypt(e, key)
		if err != nil {
			t.Errorf("failed decrypt case=%d: %e", i, err)
		}
		if !bytes.Equal(d, cb) {
			t.Errorf("failed compare decrypt case=%d", i)
		}
	}
}

func TestDecrypt(t *testing.T) {
	key := buildKey([]byte("abc"))
	// first 16 bytes is IV
	cases := []string{
		"                text",
		"                other text",
		"                other long long text",
	}
	for i, c := range cases {
		cb := []byte(c)
		d, err := Decrypt(cb, key)
		if err != nil {
			t.Errorf("failed decrypt case=%d: %e", i, err)
		}
		if n := len(d); n != len(cb)-aes.BlockSize {
			t.Errorf("unexpected lenght=%d for case=%d", n, i)
		}
		if bytes.Equal([]byte(c)[aes.BlockSize:], d) {
			t.Errorf("unexpected result for case=%d", i)
		}
	}
}

func BenchmarkEncrypt(b *testing.B) {
	key := buildKey([]byte("abc"))
	msg := []byte("some secret text")
	for n := 0; n < b.N; n++ {
		e, err := Encrypt(msg, key)
		if err != nil {
			b.Error("failed encrypt")
		}
		d, err := Decrypt(e, key)
		if err != nil {
			b.Error("failed decrypt")
		}
		if !bytes.Equal(d, msg) {
			b.Error("failed decrypted compare")
		}
	}
}
